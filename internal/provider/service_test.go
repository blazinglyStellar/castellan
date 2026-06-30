package provider

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

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

func (m *mockQuerier) CreateProvider(ctx context.Context, arg repository.CreateProviderParams) (repository.Provider, error) {
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

func (m *mockQuerier) GetProviderByID(ctx context.Context, id uuid.UUID) (repository.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.providers[id]
	if !ok {
		return repository.Provider{}, errors.New("not found")
	}
	return p, nil
}

func (m *mockQuerier) ListProvidersByOwner(ctx context.Context, ownerID uuid.UUID) ([]repository.Provider, error) {
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

func (m *mockQuerier) UpdateProvider(ctx context.Context, arg repository.UpdateProviderParams) (repository.Provider, error) {
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

func (m *mockQuerier) UpdateProviderStatus(ctx context.Context, arg repository.UpdateProviderStatusParams) (repository.Provider, error) {
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

func (m *mockQuerier) DeleteProvider(ctx context.Context, id uuid.UUID) (repository.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.providers[id]
	if !ok {
		return repository.Provider{}, errors.New("not found")
	}
	delete(m.providers, id)
	return p, nil
}

func (m *mockQuerier) GetEndpointByProviderRouteMethod(ctx context.Context, arg repository.GetEndpointByProviderRouteMethodParams) (repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.endpoints {
		if e.ProviderID == arg.ProviderID && e.Route == arg.Route && e.Method == arg.Method {
			return e, nil
		}
	}
	return repository.ApiEndpoint{}, errors.New("not found")
}

func (m *mockQuerier) CreateEndpoint(ctx context.Context, arg repository.CreateEndpointParams) (repository.ApiEndpoint, error) {
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

func (m *mockQuerier) GetEndpointByID(ctx context.Context, id uuid.UUID) (repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.endpoints[id]
	if !ok {
		return repository.ApiEndpoint{}, errors.New("not found")
	}
	return e, nil
}

func (m *mockQuerier) ListEndpointsByProvider(ctx context.Context, arg repository.ListEndpointsByProviderParams) ([]repository.ApiEndpoint, error) {
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

func (m *mockQuerier) UpdateEndpoint(ctx context.Context, arg repository.UpdateEndpointParams) (repository.ApiEndpoint, error) {
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

func (m *mockQuerier) UpdateEndpointStatus(ctx context.Context, arg repository.UpdateEndpointStatusParams) (repository.ApiEndpoint, error) {
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

func (m *mockQuerier) DeleteEndpoint(ctx context.Context, id uuid.UUID) (repository.ApiEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.endpoints[id]
	if !ok {
		return repository.ApiEndpoint{}, errors.New("not found")
	}
	delete(m.endpoints, id)
	return e, nil
}

const testRoute = "/test"

func numericFromInt64(v int64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(v), Exp: 0, Valid: true}
}

func TestCreateProvider_Success(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()

	p, err := s.CreateProvider(context.Background(), ownerID, "My Provider", "https://api.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "My Provider" {
		t.Errorf("expected name 'My Provider', got %q", p.Name)
	}
	if p.BaseUrl != "https://api.example.com" {
		t.Errorf("expected base_url 'https://api.example.com', got %q", p.BaseUrl)
	}
	if p.OwnerID != ownerID {
		t.Errorf("expected owner %s, got %s", ownerID, p.OwnerID)
	}
	if p.Status != repository.ProviderStatusActive {
		t.Errorf("expected status active, got %s", p.Status)
	}
	if p.ID == uuid.Nil {
		t.Error("expected non-zero ID")
	}
}

func TestCreateProvider_EmptyName(t *testing.T) {
	s := NewProviderService(newMockQuerier())
	_, err := s.CreateProvider(context.Background(), uuid.New(), "", "https://api.example.com")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCreateProvider_LongName(t *testing.T) {
	s := NewProviderService(newMockQuerier())
	longName := strings.Repeat("a", 256)
	_, err := s.CreateProvider(context.Background(), uuid.New(), longName, "https://api.example.com")
	if err == nil {
		t.Fatal("expected error for name > 255 chars")
	}
}

func TestCreateProvider_InvalidBaseURL_NoScheme(t *testing.T) {
	s := NewProviderService(newMockQuerier())
	_, err := s.CreateProvider(context.Background(), uuid.New(), "Test", "api.example.com")
	if err == nil {
		t.Fatal("expected error for URL without scheme")
	}
}

func TestCreateProvider_InvalidBaseURL_NotHTTP(t *testing.T) {
	s := NewProviderService(newMockQuerier())
	_, err := s.CreateProvider(context.Background(), uuid.New(), "Test", "ftp://files.example.com")
	if err == nil {
		t.Fatal("expected error for non-HTTP scheme")
	}
}

func TestCreateProvider_InvalidBaseURL_NotURL(t *testing.T) {
	s := NewProviderService(newMockQuerier())
	_, err := s.CreateProvider(context.Background(), uuid.New(), "Test", "not a url")
	if err == nil {
		t.Fatal("expected error for non-URL string")
	}
}

func TestGetProviderByID_Success(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Test Provider", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetProviderByID(context.Background(), created.ID, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, got.ID)
	}
}

func TestGetProviderByID_NotFound(t *testing.T) {
	s := NewProviderService(newMockQuerier())
	_, err := s.GetProviderByID(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent provider")
	}
}

func TestGetProviderByID_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.GetProviderByID(context.Background(), created.ID, otherUser)
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
}

func TestListProviders(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()
	otherID := uuid.New()

	s.CreateProvider(context.Background(), ownerID, "P1", "https://a.com")
	s.CreateProvider(context.Background(), ownerID, "P2", "https://b.com")
	s.CreateProvider(context.Background(), otherID, "P3", "https://c.com")

	providers, err := s.ListProviders(context.Background(), ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 2 {
		t.Errorf("expected 2 providers for owner, got %d", len(providers))
	}
}

func TestUpdateProvider_Success(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Original", "https://original.com")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := s.UpdateProvider(context.Background(), created.ID, ownerID, "Updated", "https://updated.com")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", updated.Name)
	}
	if updated.BaseUrl != "https://updated.com" {
		t.Errorf("expected base_url 'https://updated.com', got %q", updated.BaseUrl)
	}
}

func TestPartialUpdateProvider_Success(t *testing.T) {
	t.Run("update both fields", func(t *testing.T) {
		mq := newMockQuerier()
		s := NewProviderService(mq)
		ownerID := uuid.New()

		created, err := s.CreateProvider(context.Background(), ownerID, "Original", "https://original.com")
		if err != nil {
			t.Fatal(err)
		}

		name := "Updated"
		baseURL := "https://updated.com"
		updated, err := s.PartialUpdateProvider(context.Background(), created.ID, ownerID, &name, &baseURL)
		if err != nil {
			t.Fatal(err)
		}
		if updated.Name != "Updated" {
			t.Errorf("expected name %q, got %q", "Updated", updated.Name)
		}
		if updated.BaseUrl != "https://updated.com" {
			t.Errorf("expected base_url %q, got %q", "https://updated.com", updated.BaseUrl)
		}
	})

	t.Run("update name only", func(t *testing.T) {
		mq := newMockQuerier()
		s := NewProviderService(mq)
		ownerID := uuid.New()

		created, err := s.CreateProvider(context.Background(), ownerID, "Original", "https://original.com")
		if err != nil {
			t.Fatal(err)
		}

		name := "NameOnly"
		updated, err := s.PartialUpdateProvider(context.Background(), created.ID, ownerID, &name, nil)
		if err != nil {
			t.Fatal(err)
		}
		if updated.Name != "NameOnly" {
			t.Errorf("expected name %q, got %q", "NameOnly", updated.Name)
		}
		if updated.BaseUrl != "https://original.com" {
			t.Errorf("expected base_url %q (unchanged), got %q", "https://original.com", updated.BaseUrl)
		}
	})

	t.Run("update base_url only", func(t *testing.T) {
		mq := newMockQuerier()
		s := NewProviderService(mq)
		ownerID := uuid.New()

		created, err := s.CreateProvider(context.Background(), ownerID, "Original", "https://original.com")
		if err != nil {
			t.Fatal(err)
		}

		baseURL := "https://new.example.com"
		updated, err := s.PartialUpdateProvider(context.Background(), created.ID, ownerID, nil, &baseURL)
		if err != nil {
			t.Fatal(err)
		}
		if updated.Name != "Original" {
			t.Errorf("expected name %q (unchanged), got %q", "Original", updated.Name)
		}
		if updated.BaseUrl != "https://new.example.com" {
			t.Errorf("expected base_url %q, got %q", "https://new.example.com", updated.BaseUrl)
		}
	})
}

func TestPartialUpdateProvider_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	name := "Hacked"
	_, err = s.PartialUpdateProvider(context.Background(), created.ID, otherUser, &name, nil)
	if err == nil {
		t.Fatal("expected error for non-owner partial update")
	}
}

func TestPartialUpdateProvider_NotFound(t *testing.T) {
	s := NewProviderService(newMockQuerier())
	name := "Test"
	_, err := s.PartialUpdateProvider(context.Background(), uuid.New(), uuid.New(), &name, nil)
	if err == nil {
		t.Fatal("expected error for non-existent provider")
	}
}

func TestUpdateProvider_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.UpdateProvider(context.Background(), created.ID, otherUser, "Hacked", "https://evil.com")
	if err == nil {
		t.Fatal("expected error for non-owner update")
	}
}

func TestUpdateProviderStatus_Success(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := s.UpdateProviderStatus(context.Background(), created.ID, ownerID, repository.ProviderStatusInactive)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != repository.ProviderStatusInactive {
		t.Errorf("expected status inactive, got %s", updated.Status)
	}
}

func TestUpdateProviderStatus_InvalidStatus(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.UpdateProviderStatus(context.Background(), created.ID, ownerID, repository.ProviderStatus("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestUpdateProviderStatus_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.UpdateProviderStatus(context.Background(), created.ID, otherUser, repository.ProviderStatusInactive)
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
}

func TestDeleteProvider_Success(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	deleted, err := s.DeleteProvider(context.Background(), created.ID, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ID != created.ID {
		t.Errorf("expected deleted ID %s, got %s", created.ID, deleted.ID)
	}

	_, err = s.GetProviderByID(context.Background(), created.ID, ownerID)
	if err == nil {
		t.Fatal("expected error for deleted provider")
	}
}

func TestDeleteProvider_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	s := NewProviderService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	created, err := s.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.DeleteProvider(context.Background(), created.ID, otherUser)
	if err == nil {
		t.Fatal("expected error for non-owner delete")
	}
}

func TestCreateEndpoint_Success(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	input := CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       "/v1/chat",
		Method:      http.MethodPost,
		PriceAmount: numericFromInt64(100),
		Currency:    repository.CurrencyXLM,
		RateLimit:   pgtype.Int4{Int32: 10, Valid: true},
		Status:      repository.EndpointStatusActive,
	}
	endpoint, err := es.CreateEndpoint(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.Route != "/v1/chat" {
		t.Errorf("expected route '/v1/chat', got %q", endpoint.Route)
	}
	if endpoint.Method != http.MethodPost {
		t.Errorf("expected method 'POST', got %q", endpoint.Method)
	}
	if endpoint.ProviderID != provider.ID {
		t.Errorf("expected provider ID %s, got %s", provider.ID, endpoint.ProviderID)
	}
}

func TestCreateEndpoint_NotOwnerOfProvider(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	input := CreateEndpointInput{
		OwnerID:     otherUser,
		ProviderID:  provider.ID,
		Route:       "/v1/chat",
		Method:      http.MethodPost,
		PriceAmount: numericFromInt64(100),
		Currency:    repository.CurrencyXLM,
		RateLimit:   pgtype.Int4{Int32: 10, Valid: true},
		Status:      repository.EndpointStatusActive,
	}
	_, err = es.CreateEndpoint(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for non-owner creating endpoint")
	}
}

func TestCreateEndpoint_InvalidRoute(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		route string
	}{
		{"empty route", ""},
		{"no leading slash", "v1/chat"},
		{"too long", "/" + strings.Repeat("a", 2048)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CreateEndpointInput{
				OwnerID:     ownerID,
				ProviderID:  provider.ID,
				Route:       tt.route,
				Method:      http.MethodGet,
				PriceAmount: numericFromInt64(10),
				Currency:    repository.CurrencyXLM,
				Status:      repository.EndpointStatusActive,
			}
			_, err := es.CreateEndpoint(context.Background(), input)
			if err == nil {
				t.Error("expected error for invalid route")
			}
		})
	}
}

func TestCreateEndpoint_InvalidMethod(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	input := CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       testRoute,
		Method:      "INVALID",
		PriceAmount: numericFromInt64(10),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	}
	_, err = es.CreateEndpoint(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid method")
	}
}

func TestCreateEndpoint_NegativePrice(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	input := CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       testRoute,
		Method:      http.MethodGet,
		PriceAmount: pgtype.Numeric{Int: big.NewInt(-100), Exp: 0, Valid: true},
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	}
	_, err = es.CreateEndpoint(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for negative price")
	}
}

func TestCreateEndpoint_InvalidCurrency(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	input := CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       testRoute,
		Method:      http.MethodGet,
		PriceAmount: numericFromInt64(10),
		Currency:    repository.Currency("BTC"),
		Status:      repository.EndpointStatusActive,
	}
	_, err = es.CreateEndpoint(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for invalid currency")
	}
}

func TestCreateEndpoint_InvalidRateLimit(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	input := CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       testRoute,
		Method:      http.MethodGet,
		PriceAmount: numericFromInt64(10),
		Currency:    repository.CurrencyXLM,
		RateLimit:   pgtype.Int4{Int32: 0, Valid: true},
		Status:      repository.EndpointStatusActive,
	}
	_, err = es.CreateEndpoint(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for zero rate limit")
	}
}

func TestGetEndpointByID_Success(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	input := CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       testRoute,
		Method:      http.MethodGet,
		PriceAmount: numericFromInt64(50),
		Currency:    repository.CurrencyUSDC,
		Status:      repository.EndpointStatusDraft,
	}
	created, err := es.CreateEndpoint(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	got, err := es.GetEndpointByID(context.Background(), created.ID, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, got.ID)
	}
}

func TestGetEndpointByID_NotFound(t *testing.T) {
	es := NewEndpointService(newMockQuerier())
	_, err := es.GetEndpointByID(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent endpoint")
	}
}

func TestGetEndpointByID_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	input := CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       testRoute,
		Method:      http.MethodGet,
		PriceAmount: numericFromInt64(50),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	}
	created, err := es.CreateEndpoint(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	_, err = es.GetEndpointByID(context.Background(), created.ID, otherUser)
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
}

func TestListEndpoints(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	es.CreateEndpoint(context.Background(), CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: "/ep1", Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	})
	es.CreateEndpoint(context.Background(), CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: "/ep2", Method: http.MethodPost,
		PriceAmount: numericFromInt64(20), Currency: repository.CurrencyUSDC,
		RateLimit: pgtype.Int4{Int32: 5, Valid: true},
		Status:    repository.EndpointStatusActive,
	})

	endpoints, err := es.ListEndpoints(context.Background(), provider.ID, ownerID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(endpoints))
	}
}

func TestUpdateEndpoint_Success(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	createInput := CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: "/old", Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	}
	created, err := es.CreateEndpoint(context.Background(), createInput)
	if err != nil {
		t.Fatal(err)
	}

	updateInput := UpdateEndpointInput{
		OwnerID: ownerID, EndpointID: created.ID,
		Route: "/new", Method: http.MethodPost,
		PriceAmount: numericFromInt64(99), Currency: repository.CurrencyUSDC,
		RateLimit: pgtype.Int4{Int32: 20, Valid: true},
	}
	updated, err := es.UpdateEndpoint(context.Background(), updateInput)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Route != "/new" {
		t.Errorf("expected route '/new', got %q", updated.Route)
	}
	if updated.Method != http.MethodPost {
		t.Errorf("expected method 'POST', got %q", updated.Method)
	}
}

func TestUpdateEndpoint_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	createInput := CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: testRoute, Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	}
	created, err := es.CreateEndpoint(context.Background(), createInput)
	if err != nil {
		t.Fatal(err)
	}

	updateInput := UpdateEndpointInput{
		OwnerID: otherUser, EndpointID: created.ID,
		Route: "/hacked", Method: http.MethodPost,
		PriceAmount: numericFromInt64(1), Currency: repository.CurrencyXLM,
	}
	_, err = es.UpdateEndpoint(context.Background(), updateInput)
	if err == nil {
		t.Fatal("expected error for non-owner update")
	}
}

func TestUpdateEndpointStatus_Success(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	createInput := CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: testRoute, Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	}
	created, err := es.CreateEndpoint(context.Background(), createInput)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := es.UpdateEndpointStatus(context.Background(), created.ID, ownerID, repository.EndpointStatusInactive)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != repository.EndpointStatusInactive {
		t.Errorf("expected status inactive, got %s", updated.Status)
	}
}

func TestUpdateEndpointStatus_InvalidStatus(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	createInput := CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: testRoute, Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	}
	created, err := es.CreateEndpoint(context.Background(), createInput)
	if err != nil {
		t.Fatal(err)
	}

	_, err = es.UpdateEndpointStatus(context.Background(), created.ID, ownerID, repository.EndpointStatus("bogus"))
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestUpdateEndpointStatus_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	createInput := CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: testRoute, Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	}
	created, err := es.CreateEndpoint(context.Background(), createInput)
	if err != nil {
		t.Fatal(err)
	}

	_, err = es.UpdateEndpointStatus(context.Background(), created.ID, otherUser, repository.EndpointStatusInactive)
	if err == nil {
		t.Fatal("expected error for non-owner")
	}
}

func TestDeleteEndpoint_Success(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	createInput := CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: testRoute, Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	}
	created, err := es.CreateEndpoint(context.Background(), createInput)
	if err != nil {
		t.Fatal(err)
	}

	deleted, err := es.DeleteEndpoint(context.Background(), created.ID, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ID != created.ID {
		t.Errorf("expected deleted ID %s, got %s", created.ID, deleted.ID)
	}

	_, err = es.GetEndpointByID(context.Background(), created.ID, ownerID)
	if err == nil {
		t.Fatal("expected error for deleted endpoint")
	}
}

func TestDeleteEndpoint_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	createInput := CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: testRoute, Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	}
	created, err := es.CreateEndpoint(context.Background(), createInput)
	if err != nil {
		t.Fatal(err)
	}

	_, err = es.DeleteEndpoint(context.Background(), created.ID, otherUser)
	if err == nil {
		t.Fatal("expected error for non-owner delete")
	}
}

func TestPartialUpdateEndpoint_Success(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		mq := newMockQuerier()
		ps := NewProviderService(mq)
		es := NewEndpointService(mq)
		ownerID := uuid.New()

		provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
		if err != nil {
			t.Fatal(err)
		}
		created, err := es.CreateEndpoint(context.Background(), CreateEndpointInput{
			OwnerID: ownerID, ProviderID: provider.ID,
			Route: "/old", Method: http.MethodGet,
			PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
			Status: repository.EndpointStatusActive,
		})
		if err != nil {
			t.Fatal(err)
		}

		route := "/new"
		method := "POST"
		price := numericFromInt64(99)
		currency := repository.CurrencyUSDC
		rateLimit := pgtype.Int4{Int32: 20, Valid: true}

		updated, err := es.PartialUpdateEndpoint(context.Background(), created.ID, ownerID, PartialUpdateEndpointInput{
			Route: &route, Method: &method,
			PriceAmount: &price, Currency: &currency,
			RateLimit: &rateLimit,
		})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Route != "/new" {
			t.Errorf("expected route '/new', got %q", updated.Route)
		}
		if updated.Method != http.MethodPost {
			t.Errorf("expected method 'POST', got %q", updated.Method)
		}
	})

	t.Run("route only", func(t *testing.T) {
		mq := newMockQuerier()
		ps := NewProviderService(mq)
		es := NewEndpointService(mq)
		ownerID := uuid.New()

		provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
		if err != nil {
			t.Fatal(err)
		}
		created, err := es.CreateEndpoint(context.Background(), CreateEndpointInput{
			OwnerID: ownerID, ProviderID: provider.ID,
			Route: "/old", Method: http.MethodGet,
			PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
			Status: repository.EndpointStatusActive,
		})
		if err != nil {
			t.Fatal(err)
		}

		route := "/new"
		updated, err := es.PartialUpdateEndpoint(context.Background(), created.ID, ownerID, PartialUpdateEndpointInput{
			Route: &route,
		})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Route != "/new" {
			t.Errorf("expected route '/new', got %q", updated.Route)
		}
		if updated.Method != http.MethodGet {
			t.Errorf("expected method unchanged 'GET', got %q", updated.Method)
		}
		if updated.Currency != repository.CurrencyXLM {
			t.Errorf("expected currency unchanged 'XLM', got %q", updated.Currency)
		}
	})

	t.Run("method only", func(t *testing.T) {
		mq := newMockQuerier()
		ps := NewProviderService(mq)
		es := NewEndpointService(mq)
		ownerID := uuid.New()

		provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
		if err != nil {
			t.Fatal(err)
		}
		created, err := es.CreateEndpoint(context.Background(), CreateEndpointInput{
			OwnerID: ownerID, ProviderID: provider.ID,
			Route: "/test", Method: http.MethodGet,
			PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
			Status: repository.EndpointStatusActive,
		})
		if err != nil {
			t.Fatal(err)
		}

		method := "POST"
		updated, err := es.PartialUpdateEndpoint(context.Background(), created.ID, ownerID, PartialUpdateEndpointInput{
			Method: &method,
		})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Method != http.MethodPost {
			t.Errorf("expected method 'POST', got %q", updated.Method)
		}
		if updated.Route != "/test" {
			t.Errorf("expected route unchanged '/test', got %q", updated.Route)
		}
	})

	t.Run("price only", func(t *testing.T) {
		mq := newMockQuerier()
		ps := NewProviderService(mq)
		es := NewEndpointService(mq)
		ownerID := uuid.New()

		provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
		if err != nil {
			t.Fatal(err)
		}
		created, err := es.CreateEndpoint(context.Background(), CreateEndpointInput{
			OwnerID: ownerID, ProviderID: provider.ID,
			Route: "/test", Method: http.MethodGet,
			PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
			Status: repository.EndpointStatusActive,
		})
		if err != nil {
			t.Fatal(err)
		}

		price := numericFromInt64(50)
		updated, err := es.PartialUpdateEndpoint(context.Background(), created.ID, ownerID, PartialUpdateEndpointInput{
			PriceAmount: &price,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !updated.PriceAmount.Valid || updated.PriceAmount.Int.Int64() != 50 {
			t.Errorf("expected price 50, got %d", updated.PriceAmount.Int.Int64())
		}
	})
}

func TestPartialUpdateEndpoint_NotOwner(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()
	otherUser := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	created, err := es.CreateEndpoint(context.Background(), CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: "/test", Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	route := "/hacked"
	_, err = es.PartialUpdateEndpoint(context.Background(), created.ID, otherUser, PartialUpdateEndpointInput{
		Route: &route,
	})
	if err == nil {
		t.Fatal("expected error for non-owner partial update")
	}
}

func TestPartialUpdateEndpoint_NotFound(t *testing.T) {
	es := NewEndpointService(newMockQuerier())
	route := "/test"
	_, err := es.PartialUpdateEndpoint(context.Background(), uuid.New(), uuid.New(), PartialUpdateEndpointInput{
		Route: &route,
	})
	if err == nil {
		t.Fatal("expected error for non-existent endpoint")
	}
}

func TestUpdateEndpoint_DuplicateConflict(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	ep1, err := es.CreateEndpoint(context.Background(), CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: "/ep1", Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = es.CreateEndpoint(context.Background(), CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: "/ep2", Method: http.MethodGet,
		PriceAmount: numericFromInt64(20), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = es.UpdateEndpoint(context.Background(), UpdateEndpointInput{
		OwnerID: ownerID, EndpointID: ep1.ID,
		Route: "/ep2", Method: http.MethodGet,
		PriceAmount: numericFromInt64(99), Currency: repository.CurrencyXLM,
	})
	if !errors.Is(err, ErrDuplicateEndpoint) {
		t.Fatalf("expected ErrDuplicateEndpoint, got %v", err)
	}
}

func TestGetEndpointByID_OwnershipProviderDeleted(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	createInput := CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: testRoute, Method: http.MethodGet,
		PriceAmount: numericFromInt64(10), Currency: repository.CurrencyXLM,
		Status: repository.EndpointStatusActive,
	}
	created, err := es.CreateEndpoint(context.Background(), createInput)
	if err != nil {
		t.Fatal(err)
	}

	ps.DeleteProvider(context.Background(), provider.ID, ownerID)

	_, err = es.GetEndpointByID(context.Background(), created.ID, ownerID)
	if err == nil {
		t.Fatal("expected error when provider is deleted")
	}
}
