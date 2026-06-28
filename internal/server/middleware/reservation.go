package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"castellan/internal/gateway"
	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
)

const errKey = "error"

// Reservation middleware reserves funds before the upstream call, then commits on 2xx
// or releases on non-2xx. Post-response ledger ops use context.WithoutCancel.
func Reservation(ledger gateway.LedgerService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			consumer := gatewaycontext.GetConsumerInfo(r.Context())
			if consumer.ConsumerID == "" {
				slog.ErrorContext(r.Context(), "missing consumer context")
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "missing consumer context"})

				return
			}

			pricing := gatewaycontext.GetPricingInfo(r.Context())
			if pricing.EndpointID == "" {
				slog.ErrorContext(r.Context(), "missing pricing context")
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "missing pricing context"})

				return
			}

			consumerUUID, err := uuid.Parse(consumer.ConsumerID)
			if err != nil {
				slog.ErrorContext(
					r.Context(),
					"invalid consumer identity",
					slog.String("consumer_id", consumer.ConsumerID),
					slog.String("error", err.Error()),
				)
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})

				return
			}

			requestID := GetRequestID(r.Context())
			if requestID == "" {
				requestID = uuid.NewString()
			}

			if err := ledger.Reserve(r.Context(), consumerUUID, pricing.PriceAmount, requestID); err != nil {
				slog.ErrorContext(
					r.Context(),
					"reservation failed",
					slog.String("consumer_id", consumer.ConsumerID),
					slog.String("error", err.Error()),
				)
				writeJSON(r.Context(), w, http.StatusServiceUnavailable, map[string]string{errKey: "reservation failed"})

				return
			}

			rw := &responseWriter{w: w}
			next.ServeHTTP(rw, r)

			// Use a detached context so the ledger commit/release is not
			// cancelled if the client disconnects after receiving the response.
			postCtx := context.WithoutCancel(r.Context())

			if rw.statusCode >= 200 && rw.statusCode < 300 {
				if err := ledger.Commit(postCtx, requestID); err != nil {
					slog.ErrorContext(
						r.Context(),
						"failed to commit reservation",
						slog.String("request_id", requestID),
						slog.String("error", err.Error()),
					)
				}
			} else {
				if err := ledger.Release(postCtx, requestID); err != nil {
					slog.ErrorContext(
						r.Context(),
						"failed to release reservation",
						slog.String("request_id", requestID),
						slog.String("error", err.Error()),
					)
				}
			}
		})
	}
}
