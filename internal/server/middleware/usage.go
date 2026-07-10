package middleware

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"time"

	"castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const usageCaptureTimeout = 10 * time.Second

// UsageEventRepository persists usage events after upstream requests complete.
type UsageEventRepository interface {
	CreateUsageEvent(ctx context.Context, arg repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error)
}

// UsageEventRepositoryFunc is an adapter that lets a function serve as a UsageEventRepository.
type UsageEventRepositoryFunc func(ctx context.Context, arg repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error)

// CreateUsageEvent delegates to the underlying function.
func (f UsageEventRepositoryFunc) CreateUsageEvent(ctx context.Context, arg repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error) {
	return f(ctx, arg)
}

// UsageCapture middleware persists usage event data after the upstream response.
// Uses context.WithoutCancel so the write survives client disconnect.
func UsageCapture(repo UsageEventRepository, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			ctx := context.WithoutCancel(r.Context())

			consumer := gatewaycontext.GetConsumerInfo(ctx)
			pricing := gatewaycontext.GetPricingInfo(ctx)
			metrics := gatewaycontext.GetUpstreamMetrics(ctx)
			requestID := GetRequestID(ctx)

			if consumer.ConsumerID == "" {
				logger.WarnContext(ctx, "usage capture skipped: missing consumer_id")
				return
			}
			if pricing.EndpointID == "" || pricing.ProviderID == "" {
				logger.WarnContext(ctx, "usage capture skipped: missing pricing info",
					slog.String("endpoint_id", pricing.EndpointID),
					slog.String("provider_id", pricing.ProviderID),
				)
				return
			}

			consumerID, err := uuid.Parse(consumer.ConsumerID)
			if err != nil {
				logger.WarnContext(ctx, "usage capture skipped: invalid consumer_id",
					slog.String("consumer_id", consumer.ConsumerID),
					slog.String("error", err.Error()),
				)
				return
			}

			providerID, err := uuid.Parse(pricing.ProviderID)
			if err != nil {
				logger.WarnContext(ctx, "usage capture skipped: invalid provider_id",
					slog.String("provider_id", pricing.ProviderID),
					slog.String("error", err.Error()),
				)
				return
			}

			endpointID, err := uuid.Parse(pricing.EndpointID)
			if err != nil {
				logger.WarnContext(ctx, "usage capture skipped: invalid endpoint_id",
					slog.String("endpoint_id", pricing.EndpointID),
					slog.String("error", err.Error()),
				)
				return
			}

			var requestCost pgtype.Numeric
			if err := requestCost.Scan(pricing.PriceAmount.String()); err != nil {
				logger.WarnContext(ctx, "failed to convert request cost",
					slog.String("request_cost", pricing.PriceAmount.String()),
					slog.String("error", err.Error()),
				)
				return
			}

			if requestID == "" {
				requestID = uuid.NewString()
			}

			var statusCode pgtype.Int4
			if metrics.StatusCode > 0 && metrics.StatusCode <= math.MaxInt32 {
				statusCode = pgtype.Int4{Int32: int32(metrics.StatusCode), Valid: true}
			}

			var latencyMs pgtype.Int4
			if metrics.LatencyMs > 0 && metrics.LatencyMs <= math.MaxInt32 {
				latencyMs = pgtype.Int4{Int32: int32(metrics.LatencyMs), Valid: true}
			}

			var responseSize pgtype.Int4
			if metrics.ResponseSize > 0 && metrics.ResponseSize <= math.MaxInt32 {
				responseSize = pgtype.Int4{Int32: int32(metrics.ResponseSize), Valid: true}
			}

			persistCtx, cancel := context.WithTimeout(ctx, usageCaptureTimeout)
			defer cancel()

			_, err = repo.CreateUsageEvent(persistCtx, repository.CreateUsageEventParams{
				ConsumerID:   consumerID,
				ProviderID:   providerID,
				EndpointID:   endpointID,
				RequestCost:  requestCost,
				Currency:     repository.Currency(pricing.Currency),
				StatusCode:   statusCode,
				LatencyMs:    latencyMs,
				ResponseSize: responseSize,
				RequestID:    requestID,
				Status:       repository.UsageStatusCompleted,
			})
			if err != nil {
				logger.ErrorContext(persistCtx, "failed to persist usage event",
					slog.String("request_id", requestID),
					slog.String("error", err.Error()),
				)
			}
		})
	}
}
