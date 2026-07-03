//revive:disable:package-directory-mismatch
package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	rateLimitScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local cutoff = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local window = tonumber(ARGV[4])
local reset = now + window * 1000

redis.call("ZREMRANGEBYSCORE", key, "-inf", cutoff)
local current = redis.call("ZCARD", key)

if current >= limit then
	return {0, 0, reset}
end

local member = now .. ":" .. ARGV[5]
redis.call("ZADD", key, now, member)
redis.call("EXPIRE", key, window * 2)

return {1, limit - current - 1, reset}
`
	millisPerSec = 1000
	minResultLen = 3
)

// RedisRateLimiter implements the RateLimiter interface using Redis sorted sets
// with a sliding window algorithm.
type RedisRateLimiter struct {
	rdb           *redis.Client
	windowSeconds int
	script        *redis.Script
}

// NewRedisRateLimiter creates a new RedisRateLimiter.
func NewRedisRateLimiter(rdb *redis.Client, windowSeconds int) *RedisRateLimiter {
	return &RedisRateLimiter{
		rdb:           rdb,
		windowSeconds: windowSeconds,
		script:        redis.NewScript(rateLimitScript),
	}
}

func randomHex8() string {
	return uuid.New().String()[:8]
}

// Allow checks whether a consumer is allowed to make a request to the given
// endpoint, implementing a sliding window rate limit using Redis sorted sets.
func (r *RedisRateLimiter) Allow(
	ctx context.Context,
	consumerID uuid.UUID,
	endpointID uuid.UUID,
	limit int,
) (bool, int, time.Time, error) {
	if limit <= 0 {
		return true, 0, time.Time{}, nil
	}

	key := fmt.Sprintf("rl:%s:%s", consumerID, endpointID)
	now := time.Now().UnixMilli()
	cutoff := now - int64(r.windowSeconds)*millisPerSec
	member := randomHex8()

	res, err := r.script.Run(ctx, r.rdb, []string{key},
		now, cutoff, limit, r.windowSeconds, member,
	).Int64Slice()
	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("rate limit eval: %w", err)
	}
	if len(res) < minResultLen {
		return false, 0, time.Time{}, fmt.Errorf("rate limit: unexpected result length %d", len(res))
	}

	allowed := res[0] == 1
	remaining := int(res[1])
	resetAt := time.UnixMilli(res[2])

	return allowed, remaining, resetAt, nil
}
