package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/server/middleware"
)

type mockResolver struct {
	baseURL string
	err     error
}

func (m *mockResolver) ResolveBaseURL(_ context.Context, _ string) (string, error) {
	return m.baseURL, m.err
}

func TestReverseProxyForwardsRequest(t *testing.T) {
	expectedBody := `{"status":"ok"}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedBody))
	}))
	defer upstream.Close()

	resolver := &mockResolver{baseURL: upstream.URL}
	proxy := NewReverseProxy(resolver, slog.Default(), DefaultConfig())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/weather/current", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if strings.TrimSpace(string(body)) != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, strings.TrimSpace(string(body)))
	}
}

func TestReverseProxyInjectsHeaders(t *testing.T) {
	var recordedRequest *http.Request

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	resolver := &mockResolver{baseURL: upstream.URL}
	proxy := NewReverseProxy(resolver, slog.Default(), DefaultConfig())

	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/data", nil)
	request.RemoteAddr = "192.168.1.1:12345"

	ctx := middleware.RequestID()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(httptest.NewRecorder(), r)
	}))

	ctx.ServeHTTP(httptest.NewRecorder(), request)

	if recordedRequest == nil {
		t.Fatal("expected upstream to receive request")
	}

	if recordedRequest.Header.Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	if !strings.HasPrefix(recordedRequest.Header.Get("X-Forwarded-For"), "192.168.1.1:12345") {
		t.Errorf("expected X-Forwarded-For to start with 192.168.1.1:12345, got %q",
			recordedRequest.Header.Get("X-Forwarded-For"))
	}
}

func TestReverseProxyInjectsConsumerHeader(t *testing.T) {
	var recordedRequest *http.Request

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	resolver := &mockResolver{baseURL: upstream.URL}
	proxy := NewReverseProxy(resolver, slog.Default(), DefaultConfig())

	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/data", nil)
	ctx := gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID: "consumer-42",
	})
	request = request.WithContext(ctx)

	proxy.ServeHTTP(httptest.NewRecorder(), request)

	if recordedRequest == nil {
		t.Fatal("expected upstream to receive request")
	}

	if got := recordedRequest.Header.Get("X-Castellan-Consumer"); got != "consumer-42" {
		t.Errorf("expected X-Castellan-Consumer header to be consumer-42, got %q", got)
	}
}

func TestReverseProxyNon2xxProxiedAsIs(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer upstream.Close()

	resolver := &mockResolver{baseURL: upstream.URL}
	proxy := NewReverseProxy(resolver, slog.Default(), DefaultConfig())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/resource", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	if !strings.Contains(string(body), "not found") {
		t.Fatalf("expected body to contain error message, got %q", string(body))
	}
}

func TestReverseProxyResolverErrorReturns502(t *testing.T) {
	resolver := &mockResolver{err: errors.New("provider not found")}
	proxy := NewReverseProxy(resolver, slog.Default(), DefaultConfig())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/resource", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, resp.StatusCode)
	}

	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}

	if payload["error"] != "invalid provider" {
		t.Fatalf("expected error message 'invalid provider', got %q", payload["error"])
	}
}

func TestReverseProxyMetricsStoredInContext(t *testing.T) {
	expectedBody := `{"temperature":25}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedBody))
	}))
	defer upstream.Close()

	resolver := &mockResolver{baseURL: upstream.URL}
	proxy := NewReverseProxy(resolver, slog.Default(), DefaultConfig())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/weather", nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)

		metrics := gatewaycontext.GetUpstreamMetrics(r.Context())

		if metrics.StatusCode != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, metrics.StatusCode)
		}

		if metrics.ResponseSize <= 0 {
			t.Errorf("expected response size > 0, got %d", metrics.ResponseSize)
		}

		if metrics.LatencyMs <= 0 {
			t.Errorf("expected latency > 0ms, got %d", metrics.LatencyMs)
		}
	})

	handler.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if strings.TrimSpace(string(body)) != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, strings.TrimSpace(string(body)))
	}
}

func TestReverseProxyLargePayload(t *testing.T) {
	largeBody := strings.Repeat("A", 11*1024*1024)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(largeBody))
	}))
	defer upstream.Close()

	resolver := &mockResolver{baseURL: upstream.URL}
	proxy := NewReverseProxy(resolver, slog.Default(), DefaultConfig())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/large", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if len(body) != len(largeBody) {
		t.Fatalf("expected response body of %d bytes, got %d", len(largeBody), len(body))
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestCountingReadCloser(t *testing.T) {
	body := io.NopCloser(strings.NewReader("test data"))
	var counter int64

	crc := &countingReadCloser{
		ReadCloser: body,
		counter:    &counter,
	}

	result, err := io.ReadAll(crc)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}

	if string(result) != "test data" {
		t.Fatalf("expected 'test data', got %q", string(result))
	}

	if counter != 9 {
		t.Fatalf("expected counter 9, got %d", counter)
	}
}

func TestParseProviderPath(t *testing.T) {
	tests := []struct {
		path           string
		wantProviderID string
		wantRest       string
		wantOK         bool
	}{
		{"/v1/providers/uuid-123/weather", "uuid-123", "/weather", true},
		{"/v1/providers/uuid-123", "uuid-123", "/", true},
		{"/v1/providers/uuid-123/", "uuid-123", "/", true},
		{"/v1/providers/uuid-123/weather/current", "uuid-123", "/weather/current", true},
		{"/health", "", "", false},
		{"/", "", "", false},
		{"/v1/providers/", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotProviderID, gotRest, gotOK := parseProviderPath(tt.path)

			if gotProviderID != tt.wantProviderID {
				t.Errorf("expected providerID %q, got %q", tt.wantProviderID, gotProviderID)
			}

			if gotRest != tt.wantRest {
				t.Errorf("expected rest %q, got %q", tt.wantRest, gotRest)
			}

			if gotOK != tt.wantOK {
				t.Errorf("expected ok %v, got %v", tt.wantOK, gotOK)
			}
		})
	}
}

func TestReverseProxyMissingProviderPath(t *testing.T) {
	proxy := NewReverseProxy(&mockResolver{}, slog.Default(), DefaultConfig())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/invalid/path", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, resp.StatusCode)
	}
}

func TestReverseProxyReadTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer upstream.Close()

	cfg := DefaultConfig()
	cfg.ReadTimeout = 50 * time.Millisecond
	cfg.WriteTimeout = 50 * time.Millisecond

	proxy := NewReverseProxy(&mockResolver{baseURL: upstream.URL}, slog.Default(), cfg)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/weather", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, resp.StatusCode)
	}

	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}

	if payload["error"] != "upstream request timed out" {
		t.Fatalf("expected error 'upstream request timed out', got %q", payload["error"])
	}
}

func TestRetryPolicy_ShouldRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		err        error
		want       bool
	}{
		{name: "200 OK", statusCode: http.StatusOK, want: false},
		{name: "201 Created", statusCode: http.StatusCreated, want: false},
		{name: "301 Moved", statusCode: http.StatusMovedPermanently, want: false},
		{name: "400 Bad Request", statusCode: http.StatusBadRequest, want: false},
		{name: "401 Unauthorized", statusCode: http.StatusUnauthorized, want: false},
		{name: "404 Not Found", statusCode: http.StatusNotFound, want: false},
		{name: "429 Too Many", statusCode: http.StatusTooManyRequests, want: false},
		{name: "500 Internal", statusCode: http.StatusInternalServerError, want: true},
		{name: "502 Bad Gateway", statusCode: http.StatusBadGateway, want: true},
		{name: "503 Service Unavailable", statusCode: http.StatusServiceUnavailable, want: true},
		{name: "504 Gateway Timeout", statusCode: http.StatusGatewayTimeout, want: true},
		{name: "connection refused", err: errors.New("connection refused"), want: true},
		{name: "dns lookup failed", err: errors.New("no such host"), want: true},
		{name: "tls handshake", err: errors.New("tls: handshake failure"), want: true},
	}

	policy := DefaultRetryPolicy()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := policy.ShouldRetry(tt.statusCode, tt.err)

			if got != tt.want {
				t.Errorf("ShouldRetry(%d, %v) = %v, want %v", tt.statusCode, tt.err, got, tt.want)
			}
		})
	}
}

func TestRetryPolicy_Backoff(t *testing.T) {
	t.Parallel()

	policy := RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   2 * time.Second,
	}

	for attempt := range 10 {
		t.Run(fmt.Sprintf("attempt_%d", attempt), func(t *testing.T) {
			t.Parallel()

			delay := policy.Backoff(attempt)

			if delay < 0 {
				t.Errorf("Backoff(%d) = %v, expected non-negative", attempt, delay)
			}

			if delay > policy.MaxDelay {
				t.Errorf("Backoff(%d) = %v, expected <= %v", attempt, delay, policy.MaxDelay)
			}

			expectedMax := policy.BaseDelay * (1 << attempt)
			if expectedMax > policy.MaxDelay {
				expectedMax = policy.MaxDelay
			}

			if delay > expectedMax+expectedMax/2 {
				t.Errorf("Backoff(%d) = %v, expected roughly <= %v", attempt, delay, expectedMax)
			}
		})
	}
}

func TestRetryRoundTripper_FlakyUpstreamFailsThenSucceeds(t *testing.T) {
	var callCount int
	const failUntil = 2

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount <= failUntil {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"upstream down"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}
	}))
	defer upstream.Close()

	cfg := DefaultConfig()
	cfg.RetryBaseDelay = 1 * time.Millisecond
	cfg.RetryMaxDelay = 5 * time.Millisecond

	proxy := NewReverseProxy(&mockResolver{baseURL: upstream.URL}, slog.Default(), cfg)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/resource", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d after retries, got %d", http.StatusOK, resp.StatusCode)
	}

	if !strings.Contains(string(body), "ok") {
		t.Fatalf("expected success body, got %q", string(body))
	}

	if callCount != failUntil+1 {
		t.Fatalf("expected %d total calls (2 failures + 1 success), got %d", failUntil+1, callCount)
	}
}

func TestRetryRoundTripper_ExhaustionReturnsLastFailure(t *testing.T) {
	var callCount int

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"always down"}`))
	}))
	defer upstream.Close()

	cfg := DefaultConfig()
	cfg.RetryMaxRetries = 2
	cfg.RetryBaseDelay = 1 * time.Millisecond
	cfg.RetryMaxDelay = 5 * time.Millisecond

	proxy := NewReverseProxy(&mockResolver{baseURL: upstream.URL}, slog.Default(), cfg)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/resource", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d after exhaustion, got %d", http.StatusServiceUnavailable, resp.StatusCode)
	}

	if !strings.Contains(string(body), "always down") {
		t.Fatalf("expected failure body, got %q", string(body))
	}

	expectedCalls := cfg.RetryMaxRetries + 1
	if callCount != expectedCalls {
		t.Fatalf("expected %d total calls (%d retries + 1 original), got %d",
			expectedCalls, cfg.RetryMaxRetries, callCount)
	}
}

func TestRetryRoundTripper_NoRetryOn4xx(t *testing.T) {
	var callCount int

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer upstream.Close()

	cfg := DefaultConfig()
	cfg.RetryBaseDelay = 1 * time.Millisecond
	cfg.RetryMaxDelay = 5 * time.Millisecond

	proxy := NewReverseProxy(&mockResolver{baseURL: upstream.URL}, slog.Default(), cfg)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/resource", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	if callCount != 1 {
		t.Fatalf("expected 1 call (no retries on 4xx), got %d", callCount)
	}
}

func TestRetryRoundTripper_NoRetryOnNonIdempotent(t *testing.T) {
	var callCount int

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"bad gateway"}`))
	}))
	defer upstream.Close()

	cfg := DefaultConfig()
	cfg.RetryBaseDelay = 1 * time.Millisecond
	cfg.RetryMaxDelay = 5 * time.Millisecond

	proxy := NewReverseProxy(&mockResolver{baseURL: upstream.URL}, slog.Default(), cfg)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/providers/uuid-123/resource", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	resp.Body.Close()

	if callCount != 1 {
		t.Fatalf("expected 1 call (no retries on non-idempotent), got %d", callCount)
	}
}

func TestRetryRoundTripper_ContextCancelledBeforeFirstAttempt(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := DefaultConfig()

	proxy := NewReverseProxy(&mockResolver{baseURL: upstream.URL}, slog.Default(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/resource", nil)
	request = request.WithContext(ctx)

	recorder := httptest.NewRecorder()
	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status %d on cancelled context, got %d", http.StatusBadGateway, resp.StatusCode)
	}
}

func TestIsIdempotent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method string
		want   bool
	}{
		{http.MethodGet, true},
		{http.MethodHead, true},
		{http.MethodPut, true},
		{http.MethodDelete, true},
		{http.MethodOptions, true},
		{http.MethodTrace, true},
		{http.MethodPost, false},
		{http.MethodPatch, false},
		{http.MethodConnect, false},
		{"INVALID", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()

			got := isIdempotent(tt.method)

			if got != tt.want {
				t.Errorf("isIdempotent(%q) = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestReverseProxyConnectTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ConnectTimeout = 1 * time.Millisecond

	proxy := NewReverseProxy(&mockResolver{baseURL: "http://192.0.2.1:9"}, slog.Default(), cfg)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/uuid-123/resource", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, resp.StatusCode)
	}

	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}

	if payload["error"] != "upstream request timed out" {
		t.Fatalf("expected error 'upstream request timed out', got %q", payload["error"])
	}
}
