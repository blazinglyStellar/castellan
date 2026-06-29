package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"castellan/internal/auth"
	"castellan/internal/gateway"
	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/proxy"
	"castellan/internal/repository/db"
	"castellan/internal/server/middleware"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type testKeyValidator struct {
	key repository.ApiKey
	err error
}

func (m *testKeyValidator) ValidateKey(_ context.Context, _ string) (repository.ApiKey, error) {
	return m.key, m.err
}

var _ middleware.KeyValidator = (*testKeyValidator)(nil)

type testSessionValidator struct {
	token *repository.SessionToken
	err   error
}

func (m *testSessionValidator) ValidateSession(_ context.Context, _ string) (*repository.SessionToken, error) {
	return m.token, m.err
}

var _ middleware.SessionValidator = (*testSessionValidator)(nil)

type mockResolver struct {
	baseURL string
	err     error
}

func (m *mockResolver) ResolveBaseURL(_ context.Context, _ string) (string, error) {
	return m.baseURL, m.err
}

func TestHandler(t *testing.T) {
	s := &Server{}
	server := httptest.NewServer(http.HandlerFunc(s.HelloWorldHandler))
	defer server.Close()
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("error making request to server. Err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", resp.Status)
	}
	expected := "{\"message\":\"Hello World\"}"
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading response body. Err: %v", err)
	}
	if expected != string(body) {
		t.Errorf("expected response body to be %v; got %v", expected, string(body))
	}
}

func TestGatewayChain_Success(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer upstream.Close()

	s := newTestServer(upstream.URL, decimal.NewFromFloat(100))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")
	req = req.WithContext(
		gatewaycontext.SetPricingInfo(
			gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
				ConsumerID: uuid.NewString(),
			}),
			gatewaycontext.PricingInfo{
				EndpointID:  uuid.NewString(),
				ProviderID:  uuid.NewString(),
				PriceAmount: decimal.NewFromFloat(1),
				Currency:    gatewaycontext.CurrencyXLM,
			},
		),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayChain_InsufficientBalance(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s := newTestServer(upstream.URL, decimal.NewFromFloat(1))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")
	req = req.WithContext(
		gatewaycontext.SetPricingInfo(
			gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
				ConsumerID: uuid.NewString(),
			}),
			gatewaycontext.PricingInfo{
				EndpointID:  uuid.NewString(),
				ProviderID:  uuid.NewString(),
				PriceAmount: decimal.NewFromFloat(10),
				Currency:    gatewaycontext.CurrencyXLM,
			},
		),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayChain_MissingConsumerContext(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s := newTestServer(upstream.URL, decimal.NewFromFloat(100))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayChain_MissingPricingContext(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s := newTestServer(upstream.URL, decimal.NewFromFloat(100))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID: uuid.NewString(),
		}),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func newTestServer(upstreamURL string, balance decimal.Decimal) *Server {
	resolver := &mockResolver{baseURL: upstreamURL}
	pxy := proxy.NewReverseProxy(resolver, slog.Default(), proxy.DefaultConfig())

	keyValidator := &testKeyValidator{
		key: repository.ApiKey{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			KeyHash:   auth.HashKey("ca_test-key"),
			Status:    repository.ApiKeyStatusActive,
			CreatedAt: time.Now().UTC(),
		},
	}

	return &Server{
		proxy: pxy,
		balance: middleware.BalanceCheckerFunc(func(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
			return balance, nil
		}),
		usageRepo: middleware.UsageEventRepositoryFunc(func(_ context.Context, _ repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error) {
			return repository.CreateUsageEventRow{}, nil
		}),
		ledger:           gateway.NoopLedger{},
		keyValidator:     keyValidator,
		sessionValidator: &testSessionValidator{},
	}
}
