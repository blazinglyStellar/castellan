package middleware

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"castellan/internal/gateway"
	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
)

// RateLimitCheck enforces per-endpoint rate limits using the provided limiter.
// It reads ConsumerInfo and RateLimitInfo from the request context and denies
// the request with 429 when the limit is exceeded.
//
// Position in the middleware chain: after PricingResolver, before BalanceCheck.
func RateLimitCheck(limiter gateway.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			consumer := gatewaycontext.GetConsumerInfo(r.Context())
			if consumer.ConsumerID == "" {
				slog.ErrorContext(r.Context(), "missing consumer context for rate limit")
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "missing consumer context"})
				return
			}

			rateLimit := gatewaycontext.GetRateLimitInfo(r.Context())
			if rateLimit.MaxRequests == 0 {
				next.ServeHTTP(w, r)
				return
			}

			consumerUUID, err := uuid.Parse(consumer.ConsumerID)
			if err != nil {
				slog.ErrorContext(r.Context(),
					"invalid consumer identity for rate limit",
					slog.String("consumer_id", consumer.ConsumerID),
					slog.String("error", err.Error()),
				)
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
				return
			}

			pricing := gatewaycontext.GetPricingInfo(r.Context())
			if pricing.EndpointID == "" {
				slog.ErrorContext(r.Context(), "missing pricing context for rate limit")
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "missing pricing context"})
				return
			}

			endpointUUID, err := uuid.Parse(pricing.EndpointID)
			if err != nil {
				slog.ErrorContext(r.Context(),
					"invalid endpoint id for rate limit",
					slog.String("endpoint_id", pricing.EndpointID),
					slog.String("error", err.Error()),
				)
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "invalid endpoint id"})
				return
			}

			allowed, remaining, resetAt, err := limiter.Allow(r.Context(), consumerUUID, endpointUUID, rateLimit.MaxRequests)
			if err != nil {
				slog.ErrorContext(r.Context(),
					"rate limit check failed",
					slog.String("consumer_id", consumer.ConsumerID),
					slog.String("endpoint_id", pricing.EndpointID),
					slog.String("error", err.Error()),
				)
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "rate limit check failed"})
				return
			}

			w.Header().Set("X-Ratelimit-Limit", strconv.Itoa(rateLimit.MaxRequests))
			w.Header().Set("X-Ratelimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-Ratelimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

			if !allowed {
				retryAfter := int(time.Until(resetAt).Seconds())
				if retryAfter < 0 {
					retryAfter = 0
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				writeJSON(r.Context(), w, http.StatusTooManyRequests, map[string]string{
					errKey:        "rate_limit_exceeded",
					"reset":       strconv.FormatInt(resetAt.Unix(), 10),
					"retry_after": strconv.Itoa(retryAfter),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
