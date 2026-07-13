//go:build integration

package gateway_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"castellan/internal/gateway"

	"github.com/google/uuid"
)

func TestRedisRateLimiter_AllowWithinLimit(t *testing.T) {
	limiter := gateway.NewRedisRateLimiter(testRdb, 60)
	consumerID := uuid.New()
	endpointID := uuid.New()

	for i := 0; i < 5; i++ {
		allowed, remaining, resetAt, err := limiter.Allow(context.Background(), consumerID, endpointID, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
		expected := 5 - i - 1
		if remaining != expected {
			t.Fatalf("request %d: expected remaining %d, got %d", i+1, expected, remaining)
		}
		if resetAt.IsZero() {
			t.Fatal("resetAt should not be zero")
		}
	}
}

func TestRedisRateLimiter_DenyWhenExceeded(t *testing.T) {
	limiter := gateway.NewRedisRateLimiter(testRdb, 60)
	consumerID := uuid.New()
	endpointID := uuid.New()

	for i := 0; i < 3; i++ {
		allowed, _, _, err := limiter.Allow(context.Background(), consumerID, endpointID, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	allowed, remaining, resetAt, err := limiter.Allow(context.Background(), consumerID, endpointID, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("should be denied")
	}
	if remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", remaining)
	}
	if resetAt.IsZero() {
		t.Fatal("resetAt should not be zero")
	}
}

func TestRedisRateLimiter_AfterWindowPasses(t *testing.T) {
	limiter := gateway.NewRedisRateLimiter(testRdb, 1)
	consumerID := uuid.New()
	endpointID := uuid.New()

	allowed, _, _, err := limiter.Allow(context.Background(), consumerID, endpointID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("first request should be allowed")
	}

	time.Sleep(1100 * time.Millisecond)

	allowed, remaining, _, err := limiter.Allow(context.Background(), consumerID, endpointID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("request after window should be allowed")
	}
	if remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", remaining)
	}
}

func TestRedisRateLimiter_ZeroLimitAlwaysAllowed(t *testing.T) {
	limiter := gateway.NewRedisRateLimiter(testRdb, 60)
	consumerID := uuid.New()
	endpointID := uuid.New()

	for i := 0; i < 10; i++ {
		allowed, remaining, resetAt, err := limiter.Allow(context.Background(), consumerID, endpointID, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should be allowed with limit=0", i+1)
		}
		if remaining != 0 {
			t.Fatalf("expected remaining 0, got %d", remaining)
		}
		if !resetAt.IsZero() {
			t.Fatal("resetAt should be zero with limit=0")
		}
	}
}

func TestRedisRateLimiter_IsolatedPerKey(t *testing.T) {
	limiter := gateway.NewRedisRateLimiter(testRdb, 60)
	consumerA := uuid.New()
	consumerB := uuid.New()
	endpointID := uuid.New()

	allowed, _, _, err := limiter.Allow(context.Background(), consumerA, endpointID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("consumer A first request should be allowed")
	}

	allowed, _, _, err = limiter.Allow(context.Background(), consumerB, endpointID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("consumer B first request should be allowed (different key)")
	}

	allowed, _, _, err = limiter.Allow(context.Background(), consumerA, endpointID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("consumer A second request should be denied")
	}
}

func TestRedisRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := gateway.NewRedisRateLimiter(testRdb, 60)
	consumerID := uuid.New()
	endpointID := uuid.New()
	const limit = 10

	var (
		mu      sync.Mutex
		results = make([]bool, limit)
		errs    = make([]error, limit)
		wg      sync.WaitGroup
	)

	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			allowed, _, _, err := limiter.Allow(context.Background(), consumerID, endpointID, limit)
			mu.Lock()
			results[idx] = allowed
			errs[idx] = err
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, err)
		}
	}

	allowedCount := 0
	for _, allowed := range results {
		if allowed {
			allowedCount++
		}
	}
	if allowedCount != limit {
		t.Fatalf("expected %d allowed, got %d", limit, allowedCount)
	}

	extra, _, _, err := limiter.Allow(context.Background(), consumerID, endpointID, limit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extra {
		t.Fatal("extra request beyond limit should be denied")
	}
}

func TestRedisRateLimiter_SetsTTL(t *testing.T) {
	limiter := gateway.NewRedisRateLimiter(testRdb, 60)
	consumerID := uuid.New()
	endpointID := uuid.New()

	allowed, _, _, err := limiter.Allow(context.Background(), consumerID, endpointID, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("first request should be allowed")
	}

	key := fmt.Sprintf("rl:%s:%s", consumerID, endpointID)
	ttl, err := testRdb.TTL(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("failed to get TTL: %v", err)
	}
	if ttl <= 0 {
		t.Fatalf("expected positive TTL, got %v", ttl)
	}
	if ttl > 130*time.Second {
		t.Fatalf("expected TTL around 120s (2× window), got %v", ttl)
	}
}
