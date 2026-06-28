package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const balancePrecision = 2

// BalanceChecker fetches an account balance for a given owner.
type BalanceChecker interface {
	GetAccountBalance(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error)
}

// BalanceCheckerFunc is an adapter that lets a function serve as a BalanceChecker.
type BalanceCheckerFunc func(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error)

// GetAccountBalance delegates to the underlying function.
func (f BalanceCheckerFunc) GetAccountBalance(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error) {
	return f(ctx, ownerID)
}

// BalanceCheck middleware checks the consumer's balance against the endpoint price.
// Returns 402 Payment Required when the balance is insufficient.
func BalanceCheck(balancer BalanceChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			consumer := gatewaycontext.GetConsumerInfo(r.Context())
			if consumer.ConsumerID == "" {
				slog.ErrorContext(r.Context(), "missing consumer context")
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{"error": "missing consumer context"})

				return
			}

			pricing := gatewaycontext.GetPricingInfo(r.Context())
			if pricing.EndpointID == "" {
				slog.ErrorContext(r.Context(), "missing pricing context")
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{"error": "missing pricing context"})

				return
			}

			consumerUUID, err := uuid.Parse(consumer.ConsumerID)
			if err != nil {
				slog.ErrorContext(r.Context(),
					"invalid consumer identity",
					slog.String("consumer_id", consumer.ConsumerID),
					slog.String("error", err.Error()),
				)
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{"error": "invalid consumer identity"})

				return
			}

			balance, err := balancer.GetAccountBalance(r.Context(), consumerUUID)
			if err != nil {
				slog.ErrorContext(r.Context(),
					"balance unavailable",
					slog.String("consumer_id", consumer.ConsumerID),
					slog.String("error", err.Error()),
				)
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{"error": "balance unavailable"})

				return
			}

			if balance.Cmp(pricing.PriceAmount) < 0 {
				writeJSON(r.Context(), w, http.StatusPaymentRequired, map[string]string{
					"error":    "insufficient_balance",
					"balance":  balance.StringFixed(balancePrecision),
					"required": pricing.PriceAmount.StringFixed(balancePrecision),
				})

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(ctx context.Context, w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.ErrorContext(ctx, "failed to encode balance response",
			slog.String("error", err.Error()),
		)
	}
}
