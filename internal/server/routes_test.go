package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"castellan/internal/auth"
	"castellan/internal/gateway"
	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/provider"
	"castellan/internal/proxy"
	"castellan/internal/repository/db"
	"castellan/internal/server/middleware"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

// mockQuerier implements repository.Querier with in-memory provider/endpoint storage.
type mockQuerier struct {
	repository.Querier

	mu        sync.Mutex
	providers map[uuid.UUID]repository.Provider
	endpoints map[uuid.UUID]repository.ApiEndpoint
}

func newMockQuerier() *mockQuerier {
	return &mockQuerier{
		providers: make(map[uuid.UUID]repository.Provider),
		endpoints: make(map[uuid.UUID]repository.ApiEndpoint),
	}
}

func (m *mockQuerier) CreateProvider(_ context.Context, arg repository.CreateProviderParams) (repository.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := repository.Provider{
		ID:        uuid.New(),
		OwnerID:   arg.OwnerID,
		Name:      arg.Name,
		BaseUrl:   arg.BaseUrl,
		Status:    arg.Status,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	m.providers[p.ID] = p
	return p, nil
}

func (m *mockQuerier) GetProviderByID(_ context.Context, id uuid.UUID) (repository.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.providers[id]
	if !ok {
		return repository.Provider{}, errors.New("not found")
	}
	return p, nil
}

func (m *mockQuerier) ListProvidersByOwner(_ context.Context, ownerID uuid.UUID) ([]repository.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []repository.Provider
	for _, p := range m.providers {
		if p.OwnerID == ownerID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockQuerier) UpdateProvider(_ context.Context, arg repository.UpdateProviderParams) (repository.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.providers[arg.ID]
	if !ok {
		return repository.Provider{}, errors.New("not found")
	}
	p.Name = arg.Name
	p.BaseUrl = arg.BaseUrl
	p.UpdatedAt = time.Now().UTC()
	m.providers[arg.ID] = p
	return p, nil
}

func (m *mockQuerier) UpdateProviderStatus(_ context.Context, arg repository.UpdateProviderStatusParams) (repository.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.providers[arg.ID]
	if !ok {
		return repository.Provider{}, errors.New("not found")
	}
	p.Status = arg.Status
	p.UpdatedAt = time.Now().UTC()
	m.providers[arg.ID] = p
	return p, nil
}

func (m *mockQuerier) DeleteProvider(_ context.Context, id uuid.UUID) (repository.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.providers[id]
	if !ok {
		return repository.Provider{}, errors.New("not found")
	}
	delete(m.providers, id)
	return p, nil
}

func (m *mockQuerier) GetEndpointByProviderRouteMethod(_ context.Context, arg repository.GetEndpointByProviderRouteMethodParams) (repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.endpoints {
		if e.ProviderID == arg.ProviderID && e.Route == arg.Route && e.Method == arg.Method {
			return e, nil
		}
	}
	return repository.ApiEndpoint{}, errors.New("not found")
}

func (m *mockQuerier) CreateEndpoint(_ context.Context, arg repository.CreateEndpointParams) (repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := repository.ApiEndpoint{
		ID:          uuid.New(),
		ProviderID:  arg.ProviderID,
		Route:       arg.Route,
		Method:      arg.Method,
		PriceAmount: arg.PriceAmount,
		Currency:    arg.Currency,
		RateLimit:   arg.RateLimit,
		Status:      arg.Status,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	m.endpoints[e.ID] = e
	return e, nil
}

func (m *mockQuerier) GetEndpointByID(_ context.Context, id uuid.UUID) (repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.endpoints[id]
	if !ok {
		return repository.ApiEndpoint{}, errors.New("not found")
	}
	return e, nil
}

func (m *mockQuerier) ListEndpointsByProvider(_ context.Context, arg repository.ListEndpointsByProviderParams) ([]repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []repository.ApiEndpoint
	for _, e := range m.endpoints {
		if e.ProviderID != arg.ProviderID {
			continue
		}
		if arg.Status != nil {
			if statusPtr, ok := arg.Status.(*repository.EndpointStatus); ok {
				if string(e.Status) != string(*statusPtr) {
					continue
				}
			} else if statusStr, ok := arg.Status.(string); ok {
				if string(e.Status) != statusStr {
					continue
				}
			}
		}
		result = append(result, e)
	}
	return result, nil
}

func (m *mockQuerier) UpdateEndpoint(_ context.Context, arg repository.UpdateEndpointParams) (repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.endpoints[arg.ID]
	if !ok {
		return repository.ApiEndpoint{}, errors.New("not found")
	}
	e.Route = arg.Route
	e.Method = arg.Method
	e.PriceAmount = arg.PriceAmount
	e.Currency = arg.Currency
	e.RateLimit = arg.RateLimit
	e.UpdatedAt = time.Now().UTC()
	m.endpoints[arg.ID] = e
	return e, nil
}

func (m *mockQuerier) UpdateEndpointStatus(_ context.Context, arg repository.UpdateEndpointStatusParams) (repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.endpoints[arg.ID]
	if !ok {
		return repository.ApiEndpoint{}, errors.New("not found")
	}
	e.Status = arg.Status
	e.UpdatedAt = time.Now().UTC()
	m.endpoints[arg.ID] = e
	return e, nil
}

func (m *mockQuerier) DeleteEndpoint(_ context.Context, id uuid.UUID) (repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.endpoints[id]
	if !ok {
		return repository.ApiEndpoint{}, errors.New("not found")
	}
	delete(m.endpoints, id)
	return e, nil
}

func testPriceAmount() pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(10), Exp: 0, Valid: true}
}

const (
	testProviderName = "Test"
	testProviderURL  = "https://example.com"
)

// authenticatedRequest sets ConsumerInfo on the request context for auth checks.
func authenticatedRequest(r *http.Request, consumerID string) *http.Request {
	return r.WithContext(
		gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      consumerID,
			IsAuthenticated: true,
		}),
	)
}

func createTestProvider(t *testing.T, mq *mockQuerier, ownerID uuid.UUID, name, baseURL string) repository.Provider {
	t.Helper()
	p, err := mq.CreateProvider(context.Background(), repository.CreateProviderParams{
		OwnerID: ownerID, Name: name, BaseUrl: baseURL, Status: repository.ProviderStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func requireValidJSON(t *testing.T, body string) {
	t.Helper()
	if !json.Valid([]byte(body)) {
		t.Fatalf("response body is not valid JSON: %s", body)
	}
}

func createTestEndpoint(t *testing.T, mq *mockQuerier, providerID uuid.UUID) repository.ApiEndpoint {
	t.Helper()
	ep, err := mq.CreateEndpoint(context.Background(), repository.CreateEndpointParams{
		ProviderID: providerID, Route: "/ep1", Method: http.MethodGet,
		PriceAmount: testPriceAmount(), Currency: repository.CurrencyXLM, Status: repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ep
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

func newTestServerWithProviders() (*Server, uuid.UUID, *mockQuerier) {
	userID := uuid.New()
	mq := newMockQuerier()

	providerSvc := provider.NewProviderService(mq)
	endpointSvc := provider.NewEndpointService(mq)

	keyValidator := &testKeyValidator{
		key: repository.ApiKey{
			ID:        uuid.New(),
			UserID:    userID,
			KeyHash:   auth.HashKey("ca_test-key"),
			Status:    repository.ApiKeyStatusActive,
			CreatedAt: time.Now().UTC(),
		},
	}

	return &Server{
		balance: middleware.BalanceCheckerFunc(func(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
			return decimal.NewFromFloat(100), nil
		}),
		usageRepo: middleware.UsageEventRepositoryFunc(func(_ context.Context, _ repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error) {
			return repository.CreateUsageEventRow{}, nil
		}),
		ledger:           gateway.NoopLedger{},
		keyValidator:     keyValidator,
		sessionValidator: &testSessionValidator{},
		providerHandler:  provider.NewProviderHandler(providerSvc),
		endpointHandler:  provider.NewEndpointHandler(endpointSvc),
	}, userID, mq
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

// ---------------------------------------------------------------------------
// Provider & Endpoint route tests
// ---------------------------------------------------------------------------

func TestProviderRoutes_Unauthenticated(t *testing.T) {
	s, _, _ := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "POST /api/v1/providers", method: http.MethodPost, path: "/api/v1/providers", body: `{"name":"test","base_url":"https://example.com"}`},
		{name: "GET /api/v1/providers", method: http.MethodGet, path: "/api/v1/providers", body: ""},
		{name: "GET /api/v1/providers/{id}", method: http.MethodGet, path: "/api/v1/providers/" + uuid.NewString(), body: ""},
		{name: "PATCH /api/v1/providers/{id}", method: http.MethodPatch, path: "/api/v1/providers/" + uuid.NewString(), body: `{"name":"updated"}`},
		{name: "PATCH /api/v1/providers/{id}/status", method: http.MethodPatch, path: "/api/v1/providers/" + uuid.NewString() + "/status", body: `{"status":"inactive"}`},
		{name: "DELETE /api/v1/providers/{id}", method: http.MethodDelete, path: "/api/v1/providers/" + uuid.NewString(), body: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body == "" {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			} else {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestProviderCreateRoute_Authenticated(t *testing.T) {
	s, userID, _ := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers",
		strings.NewReader(`{"name":"Test Provider","base_url":"https://example.com"}`))
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	requireValidJSON(t, rec.Body.String())
}

func TestProviderListRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	createTestProvider(t, mq, userID, "P1", "https://p1.com")
	createTestProvider(t, mq, userID, "P2", "https://p2.com")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestProviderGetRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	created := createTestProvider(t, mq, userID, testProviderName, testProviderURL)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+created.ID.String(), nil)
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestProviderUpdateRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	created := createTestProvider(t, mq, userID, testProviderName, testProviderURL)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+created.ID.String(),
		strings.NewReader(`{"name":"Updated","base_url":"https://updated.com"}`))
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestProviderUpdateStatusRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	created := createTestProvider(t, mq, userID, testProviderName, testProviderURL)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+created.ID.String()+"/status",
		strings.NewReader(`{"status":"inactive"}`))
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestProviderDeleteRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	created := createTestProvider(t, mq, userID, testProviderName, testProviderURL)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/providers/"+created.ID.String(), nil)
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- Endpoint routes ---

func TestEndpointRoutes_Unauthenticated(t *testing.T) {
	s, _, _ := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "POST /api/v1/providers/{providerId}/endpoints", method: http.MethodPost, path: "/api/v1/providers/" + uuid.NewString() + "/endpoints", body: `{"route":"/test","method":"GET","price_amount":"0.50"}`},
		{name: "GET /api/v1/providers/{providerId}/endpoints", method: http.MethodGet, path: "/api/v1/providers/" + uuid.NewString() + "/endpoints", body: ""},
		{name: "GET /api/v1/endpoints/{id}", method: http.MethodGet, path: "/api/v1/endpoints/" + uuid.NewString(), body: ""},
		{name: "PATCH /api/v1/endpoints/{id}", method: http.MethodPatch, path: "/api/v1/endpoints/" + uuid.NewString(), body: `{"route":"/updated"}`},
		{name: "PATCH /api/v1/endpoints/{id}/status", method: http.MethodPatch, path: "/api/v1/endpoints/" + uuid.NewString() + "/status", body: `{"status":"inactive"}`},
		{name: "DELETE /api/v1/endpoints/{id}", method: http.MethodDelete, path: "/api/v1/endpoints/" + uuid.NewString(), body: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body == "" {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			} else {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestEndpointCreateRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	provider := createTestProvider(t, mq, userID, testProviderName, testProviderURL)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers/"+provider.ID.String()+"/endpoints",
		strings.NewReader(`{"route":"/test","method":"GET","price_amount":"0.50"}`))
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
	requireValidJSON(t, rec.Body.String())
}

func TestEndpointListRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	provider := createTestProvider(t, mq, userID, testProviderName, testProviderURL)
	createTestEndpoint(t, mq, provider.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+provider.ID.String()+"/endpoints", nil)
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestEndpointGetRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	provider := createTestProvider(t, mq, userID, testProviderName, testProviderURL)
	ep := createTestEndpoint(t, mq, provider.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/"+ep.ID.String(), nil)
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestEndpointUpdateRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	provider := createTestProvider(t, mq, userID, testProviderName, testProviderURL)
	ep := createTestEndpoint(t, mq, provider.ID)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/endpoints/"+ep.ID.String(),
		strings.NewReader(`{"route":"/updated"}`))
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestEndpointUpdateStatusRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	provider := createTestProvider(t, mq, userID, testProviderName, testProviderURL)
	ep := createTestEndpoint(t, mq, provider.ID)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/endpoints/"+ep.ID.String()+"/status",
		strings.NewReader(`{"status":"inactive"}`))
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestEndpointDeleteRoute_Authenticated(t *testing.T) {
	s, userID, mq := newTestServerWithProviders()
	handler := s.RegisterRoutes()

	provider := createTestProvider(t, mq, userID, testProviderName, testProviderURL)
	ep := createTestEndpoint(t, mq, provider.ID)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/endpoints/"+ep.ID.String(), nil)
	req = authenticatedRequest(req, userID.String())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
