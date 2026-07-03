package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EndpointResolution holds pricing and rate-limit data for a resolved endpoint.
type EndpointResolution struct {
	PricingInfo *gatewaycontext.PricingInfo
	RateLimit   int
}

// EndpointPricingResolver resolves endpoint pricing from the request path and method,
// returning pricing and rate-limit data to inject into the gateway request context.
type EndpointPricingResolver interface {
	ResolvePricing(ctx context.Context, providerID uuid.UUID, route, method string) (*EndpointResolution, error)
}

// EndpointPricingResolverFunc is an adapter that lets a function serve as a EndpointPricingResolver.
type EndpointPricingResolverFunc func(ctx context.Context, providerID uuid.UUID, route, method string) (*EndpointResolution, error)

// ResolvePricing delegates to the underlying function.
func (f EndpointPricingResolverFunc) ResolvePricing(ctx context.Context, providerID uuid.UUID, route, method string) (*EndpointResolution, error) {
	return f(ctx, providerID, route, method)
}

// PricingResolver middleware extracts the provider ID and endpoint path from
// POST /api/gateway/{providerID}/{endpoint...}, looks up the endpoint pricing
// from the api_endpoints table, and injects PricingInfo and RateLimitInfo into
// the request context.
//
// Missing or invalid provider returns 400. Endpoint not found returns 404.
func PricingResolver(resolver EndpointPricingResolver, windowSeconds int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providerID, endpointPath, ok := parseGatewayPath(r.URL.Path)
			if !ok {
				writeJSON(r.Context(), w, http.StatusBadRequest, map[string]string{errKey: "invalid gateway path"})
				return
			}

			providerUUID, err := uuid.Parse(providerID)
			if err != nil {
				writeJSON(r.Context(), w, http.StatusBadRequest, map[string]string{errKey: "invalid provider id"})
				return
			}

			resolution, err := resolver.ResolvePricing(r.Context(), providerUUID, endpointPath, r.Method)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					writeJSON(r.Context(), w, http.StatusNotFound, map[string]string{errKey: "endpoint not found"})
					return
				}

				slog.ErrorContext(r.Context(),
					"pricing resolution failed",
					slog.String("provider_id", providerID),
					slog.String("endpoint_path", endpointPath),
					slog.String("method", r.Method),
					slog.String("error", err.Error()),
				)
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "pricing resolution failed"})

				return
			}

			if resolution == nil || resolution.PricingInfo == nil {
				slog.ErrorContext(r.Context(),
					"pricing resolver returned nil resolution without error",
					slog.String("provider_id", providerID),
					slog.String("endpoint_path", endpointPath),
				)
				writeJSON(r.Context(), w, http.StatusInternalServerError, map[string]string{errKey: "pricing resolution failed"})

				return
			}

			ctx := gatewaycontext.SetPricingInfo(r.Context(), *resolution.PricingInfo)
			if resolution.RateLimit > 0 {
				ctx = gatewaycontext.SetRateLimitInfo(ctx, gatewaycontext.RateLimitInfo{
					MaxRequests:   resolution.RateLimit,
					WindowSeconds: windowSeconds,
				})
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// parseGatewayPath extracts the provider ID and endpoint route from
// /api/gateway/{providerID}/{endpoint...}.
func parseGatewayPath(path string) (providerID, rest string, ok bool) {
	const prefix = "/api/gateway/"
	const splitParts = 2

	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(trimmed, "/", splitParts)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}

	providerID = parts[0]

	if len(parts) > 1 {
		rest = "/" + parts[1]
	} else {
		rest = "/"
	}

	return providerID, rest, true
}
