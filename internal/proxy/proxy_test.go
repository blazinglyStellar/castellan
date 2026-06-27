package proxy

import (
	"context"
	"encoding/json"
	"errors"
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
	proxy := NewReverseProxy(resolver, slog.Default())

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
	proxy := NewReverseProxy(resolver, slog.Default())

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
	proxy := NewReverseProxy(resolver, slog.Default())

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
	proxy := NewReverseProxy(resolver, slog.Default())

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
	proxy := NewReverseProxy(resolver, slog.Default())

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
	proxy := NewReverseProxy(resolver, slog.Default())

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
	proxy := NewReverseProxy(resolver, slog.Default())

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
	proxy := NewReverseProxy(&mockResolver{}, slog.Default())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/invalid/path", nil)

	proxy.ServeHTTP(recorder, request)

	resp := recorder.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, resp.StatusCode)
	}
}
