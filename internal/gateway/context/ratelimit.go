//revive:disable:package-directory-mismatch
package gatewaycontext

import stdcontext "context"

type rateLimitInfoKey struct{}

// RateLimitInfo describes the rate limit configuration for an endpoint.
type RateLimitInfo struct {
	MaxRequests   int
	WindowSeconds int
}

// SetRateLimitInfo stores rate limit information in the request context.
func SetRateLimitInfo(ctx stdcontext.Context, info RateLimitInfo) stdcontext.Context {
	return stdcontext.WithValue(ctx, rateLimitInfoKey{}, info)
}

// GetRateLimitInfo reads rate limit information from the request context.
func GetRateLimitInfo(ctx stdcontext.Context) RateLimitInfo {
	info, ok := ctx.Value(rateLimitInfoKey{}).(RateLimitInfo)
	if !ok {
		return RateLimitInfo{}
	}

	return info
}
