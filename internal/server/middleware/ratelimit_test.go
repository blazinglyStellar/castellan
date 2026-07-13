package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
)

type mockRateLimiter struct {
	allowed   bool
	remaining int
	resetAt   time.Time
	err       error
}

func (m *mockRateLimiter) Allow(_ context.Context, _, _ uuid.UUID, _ int) (bool, int, time.Time, error) {
	return m.allowed, m.remaining, m.resetAt, m.err
}

func TestRateLimitCheck_WithinLimit(t *testing.T) {
	consumerID := uuid.New()
	resetAt := time.Now().Add(30 * time.Second)

	limiter := &mockRateLimiter{
		allowed:   true,
		remaining: 4,
		resetAt:   resetAt,
	}

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/v1/chat", nil)
	ctx := gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      consumerID.String(),
		IsAuthenticated: true,
	})
	ctx = gatewaycontext.SetPricingInfo(ctx, gatewaycontext.PricingInfo{
		EndpointID: uuid.New().String(),
	})
	ctx = gatewaycontext.SetRateLimitInfo(ctx, gatewaycontext.RateLimitInfo{
		MaxRequests:   5,
		WindowSeconds: 60,
	})
	request = request.WithContext(ctx)

	RateLimitCheck(limiter)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", recorder.Code)
	}
	if !called {
		t.Fatal("downstream handler should have been called")
	}

	if v := recorder.Header().Get("X-Ratelimit-Limit"); v != "5" {
		t.Fatalf("expected X-RateLimit-Limit 5, got %q", v)
	}
	if v := recorder.Header().Get("X-Ratelimit-Remaining"); v != "4" {
		t.Fatalf("expected X-RateLimit-Remaining 4, got %q", v)
	}
	if v := recorder.Header().Get("X-Ratelimit-Reset"); v == "" {
		t.Fatal("expected X-RateLimit-Reset header")
	}
}

func TestRateLimitCheck_Exceeded(t *testing.T) {
	consumerID := uuid.New()
	resetAt := time.Now().Add(30 * time.Second)

	limiter := &mockRateLimiter{
		allowed:   false,
		remaining: 0,
		resetAt:   resetAt,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("downstream handler should not be called")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/v1/chat", nil)
	ctx := gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      consumerID.String(),
		IsAuthenticated: true,
	})
	ctx = gatewaycontext.SetPricingInfo(ctx, gatewaycontext.PricingInfo{
		EndpointID: uuid.New().String(),
	})
	ctx = gatewaycontext.SetRateLimitInfo(ctx, gatewaycontext.RateLimitInfo{
		MaxRequests:   5,
		WindowSeconds: 60,
	})
	request = request.WithContext(ctx)

	RateLimitCheck(limiter)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429; got %d", recorder.Code)
	}

	if v := recorder.Header().Get("Retry-After"); v == "" {
		t.Error("expected Retry-After header on 429")
	}
	if v := recorder.Header().Get("X-Ratelimit-Limit"); v == "" {
		t.Error("expected X-RateLimit-Limit header")
	}
	if v := recorder.Header().Get("X-Ratelimit-Remaining"); v == "" {
		t.Error("expected X-RateLimit-Remaining header")
	}
	if v := recorder.Header().Get("X-Ratelimit-Reset"); v == "" {
		t.Error("expected X-RateLimit-Reset header")
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON body: %v", err)
	}
	if body["error"] != "rate_limit_exceeded" {
		t.Errorf("expected error 'rate_limit_exceeded', got %q", body["error"])
	}
	if body["reset"] == "" {
		t.Error("expected non-empty reset field in body")
	}
	if body["retry_after"] == "" {
		t.Error("expected non-empty retry_after field in body")
	}
}

func TestRateLimitCheck_NoLimitConfigured(t *testing.T) {
	consumerID := uuid.New()

	limiter := &mockRateLimiter{
		allowed:   true,
		remaining: 0,
	}

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/v1/chat", nil)
	ctx := gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      consumerID.String(),
		IsAuthenticated: true,
	})
	ctx = gatewaycontext.SetPricingInfo(ctx, gatewaycontext.PricingInfo{
		EndpointID: uuid.New().String(),
	})
	ctx = gatewaycontext.SetRateLimitInfo(ctx, gatewaycontext.RateLimitInfo{
		MaxRequests:   0,
		WindowSeconds: 60,
	})
	request = request.WithContext(ctx)

	RateLimitCheck(limiter)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", recorder.Code)
	}
	if !called {
		t.Fatal("downstream handler should have been called")
	}
}

func TestRateLimitCheck_MissingConsumer(t *testing.T) {
	limiter := &mockRateLimiter{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("downstream handler should not be called")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/v1/chat", nil)

	RateLimitCheck(limiter)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500; got %d", recorder.Code)
	}
}

func TestRateLimitCheck_MissingPricing(t *testing.T) {
	consumerID := uuid.New()

	limiter := &mockRateLimiter{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("downstream handler should not be called")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/v1/chat", nil)
	ctx := gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      consumerID.String(),
		IsAuthenticated: true,
	})
	ctx = gatewaycontext.SetRateLimitInfo(ctx, gatewaycontext.RateLimitInfo{
		MaxRequests:   5,
		WindowSeconds: 60,
	})
	request = request.WithContext(ctx)

	RateLimitCheck(limiter)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500; got %d", recorder.Code)
	}
}

func TestRateLimitCheck_LimiterError(t *testing.T) {
	consumerID := uuid.New()

	limiter := &mockRateLimiter{
		err: context.DeadlineExceeded,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("downstream handler should not be called")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/v1/chat", nil)
	ctx := gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      consumerID.String(),
		IsAuthenticated: true,
	})
	ctx = gatewaycontext.SetPricingInfo(ctx, gatewaycontext.PricingInfo{
		EndpointID: uuid.New().String(),
	})
	ctx = gatewaycontext.SetRateLimitInfo(ctx, gatewaycontext.RateLimitInfo{
		MaxRequests:   5,
		WindowSeconds: 60,
	})
	request = request.WithContext(ctx)

	RateLimitCheck(limiter)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500; got %d", recorder.Code)
	}
}
