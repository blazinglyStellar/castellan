//revive:disable:package-directory-mismatch
package gateway

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RateLimiter checks whether a consumer is allowed to make a request to an
// endpoint given the endpoint's configured rate limit.
type RateLimiter interface {
	Allow(ctx context.Context, consumerID uuid.UUID, endpointID uuid.UUID, limit int) (allowed bool, remaining int, resetAt time.Time, err error)
}

// RateLimiterFunc is an adapter that lets a function serve as a RateLimiter.
type RateLimiterFunc func(ctx context.Context, consumerID uuid.UUID, endpointID uuid.UUID, limit int) (bool, int, time.Time, error)

// Allow delegates to the underlying function.
func (f RateLimiterFunc) Allow(ctx context.Context, consumerID uuid.UUID, endpointID uuid.UUID, limit int) (bool, int, time.Time, error) {
	return f(ctx, consumerID, endpointID, limit)
}
