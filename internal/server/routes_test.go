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

	"castellan/internal/accounts"
	"castellan/internal/auth"
	"castellan/internal/gateway"
	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/ledger"
	"castellan/internal/provider"
	"castellan/internal/proxy"
	"castellan/internal/repository/db"
	"castellan/internal/server/middleware"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// mockQuerier implements repository.Querier with in-memory storage.
type mockQuerier struct {
	repository.Querier

	mu        sync.Mutex
	providers map[uuid.UUID]repository.Provider
	endpoints map[uuid.UUID]repository.ApiEndpoint
	users     map[uuid.UUID]repository.User
	entries   []repository.LedgerEntry
	entryIdx  int
}

func newMockQuerier() *mockQuerier {
	return &mockQuerier{
		providers: make(map[uuid.UUID]repository.Provider),
		endpoints: make(map[uuid.UUID]repository.ApiEndpoint),
		users:     make(map[uuid.UUID]repository.User),
		entries:   nil,
	}
}

func (m *mockQuerier) CreateProvider(_ context.Context, arg repository.CreateProviderParams) (repository.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := repository.Provider{
		ID:          uuid.New(),
		OwnerID:     arg.OwnerID,
		Name:        arg.Name,
		BaseUrl:     arg.BaseUrl,
		Description: arg.Description,
		Status:      arg.Status,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
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
	p.Description = arg.Description
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
	return repository.ApiEndpoint{}, pgx.ErrNoRows
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
		Description: arg.Description,
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
		if arg.Status.Valid {
			if string(e.Status) != string(arg.Status.EndpointStatus) {
				continue
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
	e.Description = arg.Description
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

func (m *mockQuerier) GetProviderStats(_ context.Context) ([]repository.GetProviderStatsRow, error) {
	return nil, nil
}

func (m *mockQuerier) GetUserByID(_ context.Context, id uuid.UUID) (repository.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return repository.User{}, pgx.ErrNoRows
	}
	return u, nil
}

// ---------------------------------------------------------------------------
// Ledger entry mock methods
// ---------------------------------------------------------------------------

func (m *mockQuerier) InsertLedgerEntry(_ context.Context, arg repository.InsertLedgerEntryParams) (repository.LedgerEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entryIdx++
	e := repository.LedgerEntry{
		ID:            uuid.New(),
		UserID:        arg.UserID,
		EntryType:     arg.EntryType,
		Amount:        arg.Amount,
		BalanceAfter:  arg.BalanceAfter,
		Currency:      arg.Currency,
		ReferenceID:   arg.ReferenceID,
		ReferenceType: arg.ReferenceType,
		Status:        arg.Status,
		Description:   arg.Description,
		CreatedAt:     time.Now().UTC(),
	}
	m.entries = append(m.entries, e)
	return e, nil
}

func (m *mockQuerier) GetLedgerEntryByID(_ context.Context, id uuid.UUID) (repository.LedgerEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.ID == id {
			return e, nil
		}
	}
	return repository.LedgerEntry{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetLedgerEntryByIDAndOwner(_ context.Context, arg repository.GetLedgerEntryByIDAndOwnerParams) (repository.LedgerEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.ID == arg.ID && e.UserID == arg.UserID {
			return e, nil
		}
	}
	return repository.LedgerEntry{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetLedgerEntryByReferenceID(_ context.Context, refID pgtype.UUID) (repository.LedgerEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.ReferenceID.Valid && e.ReferenceID.Bytes == refID.Bytes {
			return e, nil
		}
	}
	return repository.LedgerEntry{}, errors.New("not found")
}

func (m *mockQuerier) ListLedgerEntriesByAccount(_ context.Context, arg repository.ListLedgerEntriesByAccountParams) ([]repository.LedgerEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []repository.LedgerEntry
	for i := len(m.entries) - 1; i >= 0; i-- {
		e := m.entries[i]
		if e.UserID == arg.UserID {
			result = append(result, e)
		}
	}
	start := int(arg.Offset)
	if start > len(result) {
		start = len(result)
	}
	end := start + int(arg.Limit)
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], nil
}

func (m *mockQuerier) ListLedgerEntriesByAccountAndType(_ context.Context, arg repository.ListLedgerEntriesByAccountAndTypeParams) ([]repository.LedgerEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []repository.LedgerEntry
	for i := len(m.entries) - 1; i >= 0; i-- {
		e := m.entries[i]
		if e.UserID == arg.UserID && e.EntryType == arg.EntryType {
			result = append(result, e)
		}
	}
	start := int(arg.Offset)
	if start > len(result) {
		start = len(result)
	}
	end := start + int(arg.Limit)
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], nil
}

func (m *mockQuerier) CountLedgerEntriesByAccount(_ context.Context, userID uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, e := range m.entries {
		if e.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockQuerier) CountLedgerEntriesByAccountAndType(_ context.Context, arg repository.CountLedgerEntriesByAccountAndTypeParams) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, e := range m.entries {
		if e.UserID == arg.UserID && e.EntryType == arg.EntryType {
			count++
		}
	}
	return count, nil
}

func (m *mockQuerier) UpdateLedgerEntryStatus(_ context.Context, arg repository.UpdateLedgerEntryStatusParams) (repository.LedgerEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, e := range m.entries {
		if e.ID == arg.ID {
			e.Status = arg.Status
			m.entries[i] = e
			return e, nil
		}
	}
	return repository.LedgerEntry{}, errors.New("not found")
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
		OwnerID: ownerID, Name: name, BaseUrl: baseURL, Description: "", Status: repository.ProviderStatusActive,
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

// ---------------------------------------------------------------------------
// mockLedgerService implements gateway.LedgerService with in-memory state
// and optionally syncs balance changes to a mockQuerier.
// ---------------------------------------------------------------------------

type mockLedgerService struct {
	mu           sync.Mutex
	consumerID   uuid.UUID
	balance      decimal.Decimal
	reservations map[string]decimal.Decimal
	mq           *mockQuerier
}

func newMockLedgerService(consumerID uuid.UUID, initialBalance decimal.Decimal, mq *mockQuerier) *mockLedgerService {
	return &mockLedgerService{
		consumerID:   consumerID,
		balance:      initialBalance,
		reservations: make(map[string]decimal.Decimal),
		mq:           mq,
	}
}

func (m *mockLedgerService) Reserve(_ context.Context, consumerID uuid.UUID, amount decimal.Decimal, referenceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if consumerID != m.consumerID {
		return errors.New("unknown consumer")
	}
	if m.balance.LessThan(amount) {
		return ledger.ErrInsufficientBalance
	}
	m.balance = m.balance.Sub(amount)
	m.reservations[referenceID] = amount
	if m.mq != nil {
		m.mq.mu.Lock()
		if u, ok := m.mq.users[consumerID]; ok {
			bigFloat := new(big.Float).SetFloat64(m.balance.InexactFloat64())
			intVal, _ := bigFloat.Int(nil)
			u.Balance = pgtype.Numeric{Int: intVal, Exp: 0, Valid: true}
			m.mq.users[consumerID] = u
		}
		m.mq.mu.Unlock()
	}
	return nil
}

func (m *mockLedgerService) Commit(_ context.Context, referenceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.reservations[referenceID]; !ok {
		return ledger.ErrReservationNotFound
	}
	delete(m.reservations, referenceID)
	return nil
}

func (m *mockLedgerService) Release(_ context.Context, referenceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	amount, ok := m.reservations[referenceID]
	if !ok {
		return ledger.ErrReservationNotFound
	}
	m.balance = m.balance.Add(amount)
	delete(m.reservations, referenceID)
	if m.mq != nil {
		m.mq.mu.Lock()
		if u, ok := m.mq.users[m.consumerID]; ok {
			bigFloat := new(big.Float).SetFloat64(m.balance.InexactFloat64())
			intVal, _ := bigFloat.Int(nil)
			u.Balance = pgtype.Numeric{Int: intVal, Exp: 0, Valid: true}
			m.mq.users[m.consumerID] = u
		}
		m.mq.mu.Unlock()
	}
	return nil
}

func (m *mockLedgerService) GetBalance() decimal.Decimal {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.balance
}

// ---------------------------------------------------------------------------
// User / entry test helpers
// ---------------------------------------------------------------------------

func seedUser(t *testing.T, mq *mockQuerier, userID uuid.UUID, balance decimal.Decimal) {
	t.Helper()
	bigFloat := new(big.Float).SetFloat64(balance.InexactFloat64())
	intVal, _ := bigFloat.Int(nil)
	num := pgtype.Numeric{Int: intVal, Exp: 0, Valid: true}
	u := repository.User{
		ID:               userID,
		Balance:          num,
		Currency:         repository.CurrencyXLM,
		AccountUpdatedAt: time.Now().UTC(),
	}
	mq.mu.Lock()
	mq.users[userID] = u
	mq.mu.Unlock()
}

func seedEntry(t *testing.T, mq *mockQuerier, userID uuid.UUID, entryType repository.EntryType, amount decimal.Decimal) repository.LedgerEntry {
	t.Helper()
	bigFloat := new(big.Float).SetFloat64(amount.InexactFloat64())
	intVal, _ := bigFloat.Int(nil)
	amountNum := pgtype.Numeric{Int: intVal, Exp: 0, Valid: true}
	e, err := mq.InsertLedgerEntry(context.Background(), repository.InsertLedgerEntryParams{
		UserID:       userID,
		EntryType:    entryType,
		Amount:       amountNum,
		BalanceAfter: amountNum,
		Currency:     repository.CurrencyXLM,
		Status:       repository.LedgerStatusCompleted,
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
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

	s, mq := newTestServer(upstream.URL, decimal.NewFromFloat(100))

	providerID := uuid.New()
	_, err := mq.CreateEndpoint(context.Background(), repository.CreateEndpointParams{
		ProviderID: providerID,
		Route:      "/echo", Method: http.MethodPost,
		PriceAmount: testPriceAmount(),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+providerID.String()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")

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

	s, mq := newTestServer(upstream.URL, decimal.NewFromFloat(1))

	providerID := uuid.New()
	_, err := mq.CreateEndpoint(context.Background(), repository.CreateEndpointParams{
		ProviderID: providerID,
		Route:      "/echo", Method: http.MethodPost,
		PriceAmount: testPriceAmount(),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+providerID.String()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")

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

	s, _ := newTestServer(upstream.URL, decimal.NewFromFloat(100))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// AuthCheck passes (key is valid) but PricingResolver cannot find endpoint
	// because no endpoint was seeded → 404
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayChain_MissingPricingContext(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s, _ := newTestServer(upstream.URL, decimal.NewFromFloat(100))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// AuthCheck passes then PricingResolver cannot find the endpoint → 404
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func newTestServerWithProviders() (*Server, uuid.UUID, *mockQuerier) {
	userID := uuid.New()
	mq := newMockQuerier()

	providerSvc := provider.NewProviderService(mq)
	endpointSvc := provider.NewEndpointService(mq)
	accountSvc := accounts.NewService(mq)

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
		ledger:           &gateway.LedgerServiceFunc{},
		keyValidator:     keyValidator,
		sessionValidator: &testSessionValidator{},
		providerHandler:  provider.NewProviderHandler(providerSvc),
		endpointHandler:  provider.NewEndpointHandler(endpointSvc),
		accountHandler:   accounts.NewHandler(accountSvc),
	}, userID, mq
}

func newTestPricingResolver(mq *mockQuerier) middleware.EndpointPricingResolver {
	return middleware.EndpointPricingResolverFunc(func(ctx context.Context, providerID uuid.UUID, route, method string) (*middleware.EndpointResolution, error) {
		endpoint, err := mq.GetEndpointByProviderRouteMethod(ctx, repository.GetEndpointByProviderRouteMethodParams{
			ProviderID: providerID,
			Route:      route,
			Method:     method,
		})
		if err != nil {
			return nil, err
		}
		f64, err := endpoint.PriceAmount.Float64Value()
		if err != nil {
			return nil, err
		}
		priceAmount := decimal.NewFromFloat(f64.Float64)

		rateLimit := 0
		if endpoint.RateLimit.Valid {
			rateLimit = int(endpoint.RateLimit.Int32)
		}

		return &middleware.EndpointResolution{
			PricingInfo: &gatewaycontext.PricingInfo{
				EndpointID:  endpoint.ID.String(),
				ProviderID:  endpoint.ProviderID.String(),
				PriceAmount: priceAmount,
				Currency:    gatewaycontext.Currency(endpoint.Currency),
			},
			RateLimit: rateLimit,
		}, nil
	})
}

// newTestServerWithAccounts creates a Server wired with the full mockQuerier,
// a mockLedgerService, and the accounts handler. The consumerID is returned
// for use with authenticatedRequest.
func newTestServerWithAccounts(upstreamURL string, consumerID uuid.UUID, initialBalance decimal.Decimal) (*Server, *mockQuerier, *mockLedgerService) {
	mq := newMockQuerier()
	accountSvc := accounts.NewService(mq)

	resolver := &mockResolver{baseURL: upstreamURL}
	pxy := proxy.NewReverseProxy(resolver, slog.Default(), proxy.DefaultConfig())

	mls := newMockLedgerService(consumerID, initialBalance, mq)

	keyValidator := &testKeyValidator{
		key: repository.ApiKey{
			ID:        uuid.New(),
			UserID:    consumerID,
			KeyHash:   auth.HashKey("ca_test-key"),
			Status:    repository.ApiKeyStatusActive,
			CreatedAt: time.Now().UTC(),
		},
	}

	return &Server{
		proxy: pxy,
		balance: middleware.BalanceCheckerFunc(func(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error) {
			u, err := mq.GetUserByID(ctx, ownerID)
			if err != nil {
				return decimal.Zero, errors.New("user not found")
			}
			f64, err := u.Balance.Float64Value()
			if err != nil {
				return decimal.Zero, err
			}
			return decimal.NewFromFloat(f64.Float64), nil
		}),
		pricingResolver: newTestPricingResolver(mq),
		usageRepo: middleware.UsageEventRepositoryFunc(func(_ context.Context, _ repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error) {
			return repository.CreateUsageEventRow{}, nil
		}),
		ledger:           mls,
		keyValidator:     keyValidator,
		sessionValidator: &testSessionValidator{},
		accountHandler:   accounts.NewHandler(accountSvc),
	}, mq, mls
}

func newTestServer(upstreamURL string, balance decimal.Decimal) (*Server, *mockQuerier) {
	resolver := &mockResolver{baseURL: upstreamURL}
	pxy := proxy.NewReverseProxy(resolver, slog.Default(), proxy.DefaultConfig())
	mq := newMockQuerier()

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
		pricingResolver: newTestPricingResolver(mq),
		usageRepo: middleware.UsageEventRepositoryFunc(func(_ context.Context, _ repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error) {
			return repository.CreateUsageEventRow{}, nil
		}),
		ledger:           &gateway.LedgerServiceFunc{},
		keyValidator:     keyValidator,
		sessionValidator: &testSessionValidator{},
	}, mq
}

// ---------------------------------------------------------------------------
// Gateway ledger lifecycle tests  (Issue #108)
// ---------------------------------------------------------------------------

func TestGatewayLifecycle_BalanceDecreases(t *testing.T) {
	consumerID := uuid.New()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer upstream.Close()

	s, mq, mls := newTestServerWithAccounts(upstream.URL, consumerID, decimal.NewFromFloat(100))
	seedUser(t, mq, consumerID, decimal.NewFromFloat(100))

	providerID := uuid.New()
	_, err := mq.CreateEndpoint(context.Background(), repository.CreateEndpointParams{
		ProviderID: providerID,
		Route:      "/echo", Method: http.MethodPost,
		PriceAmount: testPriceAmount(),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+providerID.String()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	expectedBalance := decimal.NewFromFloat(90)
	if !mls.GetBalance().Equal(expectedBalance) {
		t.Fatalf("expected balance %s, got %s", expectedBalance, mls.GetBalance())
	}
}

func TestGatewayLifecycle_UpstreamFailure(t *testing.T) {
	consumerID := uuid.New()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()

	s, mq, mls := newTestServerWithAccounts(upstream.URL, consumerID, decimal.NewFromFloat(100))
	seedUser(t, mq, consumerID, decimal.NewFromFloat(100))

	providerID := uuid.New()
	_, err := mq.CreateEndpoint(context.Background(), repository.CreateEndpointParams{
		ProviderID: providerID,
		Route:      "/echo", Method: http.MethodPost,
		PriceAmount: testPriceAmount(),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+providerID.String()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	expectedBalance := decimal.NewFromFloat(100)
	if !mls.GetBalance().Equal(expectedBalance) {
		t.Fatalf("expected balance %s (released after failure), got %s", expectedBalance, mls.GetBalance())
	}
}

func TestGatewayInsufficientBalance_NoEntries(t *testing.T) {
	consumerID := uuid.New()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s, mq, _ := newTestServerWithAccounts(upstream.URL, consumerID, decimal.NewFromFloat(5))
	seedUser(t, mq, consumerID, decimal.NewFromFloat(5))

	providerID := uuid.New()
	_, err := mq.CreateEndpoint(context.Background(), repository.CreateEndpointParams{
		ProviderID: providerID,
		Route:      "/echo", Method: http.MethodPost,
		PriceAmount: testPriceAmount(),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodPost, "/api/gateway/"+providerID.String()+"/echo", nil)
	req.Header.Set("Authorization", "Bearer ca_test-key")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	total, err := mq.CountLedgerEntriesByAccount(context.Background(), consumerID)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("expected 0 ledger entries, got %d", total)
	}
}

func TestGatewayLifecycle_BalanceEndpoint(t *testing.T) {
	consumerID := uuid.New()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer upstream.Close()

	s, mq, _ := newTestServerWithAccounts(upstream.URL, consumerID, decimal.NewFromFloat(100))
	seedUser(t, mq, consumerID, decimal.NewFromFloat(100))

	providerID := uuid.New()
	_, err := mq.CreateEndpoint(context.Background(), repository.CreateEndpointParams{
		ProviderID: providerID,
		Route:      "/echo", Method: http.MethodPost,
		PriceAmount: testPriceAmount(),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := s.RegisterRoutes()

	gatewayReq := httptest.NewRequest(http.MethodPost, "/api/gateway/"+providerID.String()+"/echo", nil)
	gatewayReq.Header.Set("Authorization", "Bearer ca_test-key")

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, gatewayReq)
	if rec1.Code != http.StatusOK {
		t.Fatalf("gateway request expected 200, got %d. Body: %s", rec1.Code, rec1.Body.String())
	}

	acctReq := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me", nil)
	acctReq = authenticatedRequest(acctReq, consumerID.String())

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, acctReq)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 from account endpoint, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec2.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	balanceStr, ok := body["balance"].(string)
	if !ok {
		t.Fatalf("expected balance field as string, got %T", body["balance"])
	}
	if balanceStr != "90" {
		t.Fatalf("expected balance 90, got %s", balanceStr)
	}
}

// ---------------------------------------------------------------------------
// Account HTTP handler tests  (Issue #112)
// ---------------------------------------------------------------------------

func TestAccountHandler_GetAccount_Success(t *testing.T) {
	consumerID := uuid.New()
	s, mq, _ := newTestServerWithAccounts("http://example.com", consumerID, decimal.NewFromFloat(50))
	seedUser(t, mq, consumerID, decimal.NewFromFloat(50))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me", nil)
	req = authenticatedRequest(req, consumerID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["balance"] != "50" {
		t.Fatalf("expected balance 50, got %v", body["balance"])
	}
	if body["currency"] != "XLM" {
		t.Fatalf("expected currency XLM, got %v", body["currency"])
	}
}

func TestAccountHandler_GetAccount_Unauthenticated(t *testing.T) {
	consumerID := uuid.New()
	s, _, _ := newTestServerWithAccounts("http://example.com", consumerID, decimal.NewFromFloat(50))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me", nil)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] != "authentication required" {
		t.Fatalf("expected authentication required error, got %v", body["error"])
	}
}

func TestAccountHandler_GetAccount_NotFound(t *testing.T) {
	noAccountID := uuid.New()
	s, _, _ := newTestServerWithAccounts("http://example.com", noAccountID, decimal.NewFromFloat(0))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me", nil)
	req = authenticatedRequest(req, noAccountID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] != "account not found" {
		t.Fatalf("expected account not found error, got %v", body["error"])
	}
}

func TestAccountHandler_ListEntries_Success(t *testing.T) {
	consumerID := uuid.New()
	s, mq, _ := newTestServerWithAccounts("http://example.com", consumerID, decimal.NewFromFloat(100))
	seedUser(t, mq, consumerID, decimal.NewFromFloat(100))

	seedEntry(t, mq, consumerID, repository.EntryTypeDeduction, decimal.NewFromFloat(10))
	seedEntry(t, mq, consumerID, repository.EntryTypeDeduction, decimal.NewFromFloat(20))
	seedEntry(t, mq, consumerID, repository.EntryTypeDeposit, decimal.NewFromFloat(50))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me/entries", nil)
	req = authenticatedRequest(req, consumerID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	total, ok := body["total"].(float64)
	if !ok || int(total) != 3 {
		t.Fatalf("expected total 3, got %v", body["total"])
	}
	entries, ok := body["entries"].([]any)
	if !ok || len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestAccountHandler_ListEntries_Filtered(t *testing.T) {
	consumerID := uuid.New()
	s, mq, _ := newTestServerWithAccounts("http://example.com", consumerID, decimal.NewFromFloat(100))
	seedUser(t, mq, consumerID, decimal.NewFromFloat(100))

	seedEntry(t, mq, consumerID, repository.EntryTypeDeduction, decimal.NewFromFloat(10))
	seedEntry(t, mq, consumerID, repository.EntryTypeDeduction, decimal.NewFromFloat(20))
	seedEntry(t, mq, consumerID, repository.EntryTypeDeposit, decimal.NewFromFloat(50))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me/entries?type=deduction", nil)
	req = authenticatedRequest(req, consumerID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	total, ok := body["total"].(float64)
	if !ok || int(total) != 2 {
		t.Fatalf("expected total 2, got %v", body["total"])
	}
	entries, ok := body["entries"].([]any)
	if !ok || len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, entry := range entries {
		e, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("expected map entry, got %T", entry)
		}
		if e["entry_type"] != "deduction" {
			t.Fatalf("expected entry_type deduction, got %v", e["entry_type"])
		}
	}
}

func TestAccountHandler_GetEntry_Success(t *testing.T) {
	consumerID := uuid.New()
	s, mq, _ := newTestServerWithAccounts("http://example.com", consumerID, decimal.NewFromFloat(100))
	seedUser(t, mq, consumerID, decimal.NewFromFloat(100))

	seeded := seedEntry(t, mq, consumerID, repository.EntryTypeDeduction, decimal.NewFromFloat(10))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me/entries/"+seeded.ID.String(), nil)
	req = authenticatedRequest(req, consumerID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["entry_type"] != "deduction" {
		t.Fatalf("expected entry_type deduction, got %v", body["entry_type"])
	}
}

func TestAccountHandler_GetEntry_NotFound(t *testing.T) {
	consumerA := uuid.New()
	consumerB := uuid.New()
	s, mq, _ := newTestServerWithAccounts("http://example.com", consumerA, decimal.NewFromFloat(100))
	seedUser(t, mq, consumerA, decimal.NewFromFloat(100))

	seeded := seedEntry(t, mq, consumerA, repository.EntryTypeDeduction, decimal.NewFromFloat(10))

	handler := s.RegisterRoutes()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me/entries/"+seeded.ID.String(), nil)
	req = authenticatedRequest(req, consumerB.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
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
