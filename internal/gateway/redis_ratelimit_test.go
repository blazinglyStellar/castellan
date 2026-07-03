//go:build integration

//revive:disable:package-directory-mismatch
package gateway

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

var testRdb *goredis.Client

func mustStartRedisContainer() (func(context.Context, ...testcontainers.TerminateOption) error, error) {
	ctx := context.Background()

	container, err := tcredis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			tcwait.ForLog("* Ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("redis container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		return container.Terminate, fmt.Errorf("connection string: %w", err)
	}

	opts, err := goredis.ParseURL(connStr)
	if err != nil {
		return container.Terminate, fmt.Errorf("parse URL: %w", err)
	}

	testRdb = goredis.NewClient(opts)

	return container.Terminate, nil
}

func TestMain(m *testing.M) {
	teardown, err := mustStartRedisContainer()
	if err != nil {
		log.Fatalf("redis container: %v", err)
	}

	code := m.Run()

	if err := teardown(context.Background()); err != nil {
		log.Fatalf("teardown: %v", err)
	}

	os.Exit(code)
}

func TestRedisRateLimiter_AllowWithinLimit(t *testing.T) {
	limiter := NewRedisRateLimiter(testRdb, 60)
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
	limiter := NewRedisRateLimiter(testRdb, 60)
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
	limiter := NewRedisRateLimiter(testRdb, 1)
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
	limiter := NewRedisRateLimiter(testRdb, 60)
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
	limiter := NewRedisRateLimiter(testRdb, 60)
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
