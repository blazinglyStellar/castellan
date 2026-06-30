package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
)

func authenticatedProviderRequest(t *testing.T, method, body string) *http.Request {
	t.Helper()

	var req *http.Request
	if method == http.MethodGet {
		req = httptest.NewRequest(method, "/api/v1/providers", nil)
	} else {
		req = httptest.NewRequest(method, "/api/v1/providers", strings.NewReader(body))
	}
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)

	return req
}

func TestCreateProviderHandler_Success(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	h.CreateProvider(rec, authenticatedProviderRequest(t, http.MethodPost, `{"name":"Weather API","base_url":"https://api.weather.example.com/v2"}`))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp repository.Provider
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID == uuid.Nil {
		t.Error("expected non-nil id")
	}
	if resp.OwnerID == uuid.Nil {
		t.Error("expected non-nil owner_id")
	}
	if resp.Name != "Weather API" {
		t.Errorf("expected name %q, got %q", "Weather API", resp.Name)
	}
	if resp.BaseUrl != "https://api.weather.example.com/v2" {
		t.Errorf("expected base_url %q, got %q", "https://api.weather.example.com/v2", resp.BaseUrl)
	}
	if resp.Status != repository.ProviderStatusActive {
		t.Errorf("expected status %q, got %q", repository.ProviderStatusActive, resp.Status)
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if resp.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateProviderHandler_ValidationErrors(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))

	tests := []struct {
		name       string
		body       string
		wantErr    string
		wantStatus int
	}{
		{
			name:       "missing name",
			body:       `{"base_url":"https://example.com"}`,
			wantErr:    "name is required",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid base_url",
			body:       `{"name":"test","base_url":"not-a-url"}`,
			wantErr:    "base_url must be a valid URL",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid scheme",
			body:       `{"name":"test","base_url":"ftp://example.com"}`,
			wantErr:    "base_url scheme must be http or https",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON body",
			body:       `not json`,
			wantErr:    "invalid request body",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			var req *http.Request
			if tt.name == "invalid JSON body" {
				req = httptest.NewRequest(http.MethodPost, "/api/v1/providers", bytes.NewReader([]byte(tt.body)))
				req = req.WithContext(
					gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
						ConsumerID:      uuid.New().String(),
						IsAuthenticated: true,
					}),
				)
			} else {
				req = authenticatedProviderRequest(t, http.MethodPost, tt.body)
			}

			h.CreateProvider(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			var errResp map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if errResp["error"] != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, errResp["error"])
			}
		})
	}
}

func TestCreateProviderHandler_Unauthenticated(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers", strings.NewReader(`{"name":"test","base_url":"https://example.com"}`))
	h.CreateProvider(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListProvidersHandler_Success(t *testing.T) {
	userID := uuid.New()
	mq := newMockQuerier()
	// Seed providers
	mq.CreateProvider(context.TODO(), repository.CreateProviderParams{
		OwnerID: userID,
		Name:    "Weather API",
		BaseUrl: "https://api.weather.example.com/v2",
		Status:  repository.ProviderStatusActive,
	})
	mq.CreateProvider(context.TODO(), repository.CreateProviderParams{
		OwnerID: userID,
		Name:    "Maps API",
		BaseUrl: "https://maps.example.com",
		Status:  repository.ProviderStatusActive,
	})

	h := NewProviderHandler(NewProviderService(mq))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      userID.String(),
			IsAuthenticated: true,
		}),
	)
	h.ListProviders(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var items []repository.Provider
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(items))
	}
}

func TestListProvidersHandler_Empty(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	h.ListProviders(rec, authenticatedProviderRequest(t, http.MethodGet, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var items []repository.Provider
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if items == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 providers, got %d", len(items))
	}
}

func TestListProvidersHandler_Unauthenticated(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	h.ListProviders(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func createTestProvider(t *testing.T) (*Handler, uuid.UUID, repository.Provider) {
	t.Helper()
	mq := newMockQuerier()
	h := NewProviderHandler(NewProviderService(mq))
	userID := uuid.New()

	created, err := mq.CreateProvider(context.TODO(), repository.CreateProviderParams{
		OwnerID: userID,
		Name:    "Weather API",
		BaseUrl: "https://api.weather.example.com",
		Status:  repository.ProviderStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	return h, userID, created
}

func invalidUUIDRequest(t *testing.T, method, body string) *http.Request {
	t.Helper()

	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, "/api/v1/providers/not-a-uuid", nil)
	} else {
		req = httptest.NewRequest(method, "/api/v1/providers/not-a-uuid", strings.NewReader(body))
	}
	req.SetPathValue("id", "not-a-uuid")
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)

	return req
}

func TestGetProviderHandler_Success(t *testing.T) {
	h, userID, created := createTestProvider(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+created.ID.String(), nil)
	req.SetPathValue("id", created.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      userID.String(),
			IsAuthenticated: true,
		}),
	)
	h.GetProvider(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp repository.Provider
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, resp.ID)
	}
}

func TestGetProviderHandler_NotFound(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	id := uuid.New().String()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+id, nil)
	req.SetPathValue("id", id)
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)
	h.GetProvider(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetProviderHandler_NotOwner(t *testing.T) {
	h, _, created := createTestProvider(t)
	otherUser := uuid.New()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+created.ID.String(), nil)
	req.SetPathValue("id", created.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      otherUser.String(),
			IsAuthenticated: true,
		}),
	)
	h.GetProvider(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetProviderHandler_InvalidUUID(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	h.GetProvider(rec, invalidUUIDRequest(t, http.MethodGet, ""))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGetProviderHandler_Unauthenticated(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+uuid.New().String(), nil)
	h.GetProvider(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateProviderHandler_Success(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantName string
		wantURL  string
	}{
		{
			name:     "both fields",
			body:     `{"name":"Weather API v2","base_url":"https://api.v2.weather.example.com"}`,
			wantName: "Weather API v2",
			wantURL:  "https://api.v2.weather.example.com",
		},
		{
			name:     "name only",
			body:     `{"name":"Weather API v2"}`,
			wantName: "Weather API v2",
			wantURL:  "https://api.weather.example.com",
		},
		{
			name:     "base_url only",
			body:     `{"base_url":"https://api.v2.weather.example.com"}`,
			wantName: "Weather API",
			wantURL:  "https://api.v2.weather.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, userID, created := createTestProvider(t)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+created.ID.String(),
				strings.NewReader(tt.body))
			req.SetPathValue("id", created.ID.String())
			req = req.WithContext(
				gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
					ConsumerID:      userID.String(),
					IsAuthenticated: true,
				}),
			)
			h.UpdateProvider(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
			}

			var resp repository.Provider
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp.Name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, resp.Name)
			}
			if resp.BaseUrl != tt.wantURL {
				t.Errorf("expected base_url %q, got %q", tt.wantURL, resp.BaseUrl)
			}
		})
	}
}

func TestUpdateProviderHandler_InvalidUUID(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	h.UpdateProvider(rec, invalidUUIDRequest(t, http.MethodPatch, `{"name":"test"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateProviderHandler_Unauthenticated(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+uuid.New().String(),
		strings.NewReader(`{"name":"test"}`))
	h.UpdateProvider(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateProviderStatusHandler_Success(t *testing.T) {
	h, userID, created := createTestProvider(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+created.ID.String()+"/status",
		strings.NewReader(`{"status":"suspended"}`))
	req.SetPathValue("id", created.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      userID.String(),
			IsAuthenticated: true,
		}),
	)
	h.UpdateProviderStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp repository.Provider
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != repository.ProviderStatusSuspended {
		t.Errorf("expected status %q, got %q", repository.ProviderStatusSuspended, resp.Status)
	}
}

func TestUpdateProviderStatusHandler_InvalidStatus(t *testing.T) {
	h, userID, created := createTestProvider(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+created.ID.String()+"/status",
		strings.NewReader(`{"status":"bogus"}`))
	req.SetPathValue("id", created.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      userID.String(),
			IsAuthenticated: true,
		}),
	)
	h.UpdateProviderStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var errResp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp["error"] != "invalid provider status: bogus" {
		t.Errorf("expected error %q, got %q", "invalid provider status: bogus", errResp["error"])
	}
}

func TestUpdateProviderStatusHandler_InvalidUUID(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	h.UpdateProviderStatus(rec, invalidUUIDRequest(t, http.MethodPatch, `{"status":"active"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateProviderStatusHandler_Unauthenticated(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/"+uuid.New().String()+"/status",
		strings.NewReader(`{"status":"active"}`))
	h.UpdateProviderStatus(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProviderHandler_Success(t *testing.T) {
	h, userID, created := createTestProvider(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/providers/"+created.ID.String(), nil)
	req.SetPathValue("id", created.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      userID.String(),
			IsAuthenticated: true,
		}),
	)
	h.DeleteProvider(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProviderHandler_NotFound(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	id := uuid.New().String()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/providers/"+id, nil)
	req.SetPathValue("id", id)
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)
	h.DeleteProvider(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProviderHandler_NotOwner(t *testing.T) {
	h, _, created := createTestProvider(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/providers/"+created.ID.String(), nil)
	req.SetPathValue("id", created.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)
	h.DeleteProvider(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProviderHandler_InvalidUUID(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	h.DeleteProvider(rec, invalidUUIDRequest(t, http.MethodDelete, ""))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProviderHandler_Unauthenticated(t *testing.T) {
	h := NewProviderHandler(NewProviderService(newMockQuerier()))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/providers/"+uuid.New().String(), nil)
	h.DeleteProvider(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- Endpoint handler tests ---

func setupEndpointTest(t *testing.T) (*EndpointHandler, uuid.UUID, repository.Provider) {
	t.Helper()

	mq := newMockQuerier()
	ps := NewProviderService(mq)
	eh := NewEndpointHandler(NewEndpointService(mq))
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test Provider", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	return eh, ownerID, provider
}

func TestCreateEndpointHandler_Success(t *testing.T) {
	eh, ownerID, provider := setupEndpointTest(t)

	body := `{"route":"/current","method":"GET","price_amount":"0.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers/"+provider.ID.String()+"/endpoints", strings.NewReader(body))
	req.SetPathValue("providerId", provider.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      ownerID.String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.CreateEndpoint(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp repository.ApiEndpoint
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID == uuid.Nil {
		t.Error("expected non-nil id")
	}
	if resp.ProviderID != provider.ID {
		t.Errorf("expected provider_id %s, got %s", provider.ID, resp.ProviderID)
	}
	if resp.Route != "/current" {
		t.Errorf("expected route %q, got %q", "/current", resp.Route)
	}
	if resp.Method != http.MethodGet {
		t.Errorf("expected method %q, got %q", http.MethodGet, resp.Method)
	}
	if resp.Currency != repository.CurrencyXLM {
		t.Errorf("expected currency %q, got %q", repository.CurrencyXLM, resp.Currency)
	}
	if resp.Status != repository.EndpointStatusDraft {
		t.Errorf("expected status %q, got %q", repository.EndpointStatusDraft, resp.Status)
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if resp.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateEndpointHandler_ValidationErrors(t *testing.T) {
	eh, ownerID, provider := setupEndpointTest(t)

	tests := []struct {
		name       string
		body       string
		wantErr    string
		wantStatus int
	}{
		{
			name:       "missing route",
			body:       `{"method":"GET","price_amount":"0.50"}`,
			wantErr:    "route is required",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "route without slash",
			body:       `{"route":"current","method":"GET","price_amount":"0.50"}`,
			wantErr:    "route must start with /",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid method",
			body:       `{"route":"/current","method":"INVALID","price_amount":"0.50"}`,
			wantErr:    "invalid HTTP method: INVALID",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			body:       `{}`,
			wantErr:    "route is required",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/providers/"+provider.ID.String()+"/endpoints", strings.NewReader(tt.body))
			req.SetPathValue("providerId", provider.ID.String())
			req = req.WithContext(
				gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
					ConsumerID:      ownerID.String(),
					IsAuthenticated: true,
				}),
			)

			rec := httptest.NewRecorder()
			eh.CreateEndpoint(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			var errResp map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if errResp["error"] != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, errResp["error"])
			}
		})
	}
}

func TestCreateEndpointHandler_ProviderNotFound(t *testing.T) {
	eh := NewEndpointHandler(NewEndpointService(newMockQuerier()))
	otherUser := uuid.New()
	providerID := uuid.New()

	body := `{"route":"/current","method":"GET","price_amount":"0.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers/"+providerID.String()+"/endpoints", strings.NewReader(body))
	req.SetPathValue("providerId", providerID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      otherUser.String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.CreateEndpoint(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEndpointHandler_NotOwner(t *testing.T) {
	eh, _, provider := setupEndpointTest(t)
	otherUser := uuid.New()

	body := `{"route":"/current","method":"GET","price_amount":"0.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers/"+provider.ID.String()+"/endpoints", strings.NewReader(body))
	req.SetPathValue("providerId", provider.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      otherUser.String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.CreateEndpoint(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEndpointHandler_Duplicate(t *testing.T) {
	eh, ownerID, provider := setupEndpointTest(t)

	baseURL := "/api/v1/providers/" + provider.ID.String() + "/endpoints"

	// First create should succeed
	req1 := httptest.NewRequest(http.MethodPost, baseURL, strings.NewReader(`{"route":"/current","method":"GET","price_amount":"0.50"}`))
	req1.SetPathValue("providerId", provider.ID.String())
	req1 = req1.WithContext(
		gatewaycontext.SetConsumerInfo(req1.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      ownerID.String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.CreateEndpoint(rec, req1)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Second create with same route+method should conflict
	req2 := httptest.NewRequest(http.MethodPost, baseURL, strings.NewReader(`{"route":"/current","method":"GET","price_amount":"0.50"}`))
	req2.SetPathValue("providerId", provider.ID.String())
	req2 = req2.WithContext(
		gatewaycontext.SetConsumerInfo(req2.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      ownerID.String(),
			IsAuthenticated: true,
		}),
	)

	rec2 := httptest.NewRecorder()
	eh.CreateEndpoint(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}

	var errResp map[string]string
	if err := json.NewDecoder(rec2.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errResp["error"] != ErrDuplicateEndpoint.Error() {
		t.Errorf("expected error %q, got %q", ErrDuplicateEndpoint.Error(), errResp["error"])
	}
}

func TestCreateEndpointHandler_Unauthenticated(t *testing.T) {
	eh := NewEndpointHandler(NewEndpointService(newMockQuerier()))

	body := `{"route":"/current","method":"GET","price_amount":"0.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers/"+uuid.New().String()+"/endpoints", strings.NewReader(body))

	rec := httptest.NewRecorder()
	eh.CreateEndpoint(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEndpointHandler_InvalidProviderId(t *testing.T) {
	eh := NewEndpointHandler(NewEndpointService(newMockQuerier()))

	body := `{"route":"/current","method":"GET","price_amount":"0.50"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/providers/not-a-uuid/endpoints", strings.NewReader(body))
	req.SetPathValue("providerId", "not-a-uuid")
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.CreateEndpoint(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListEndpointsHandler_Success(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	eh := NewEndpointHandler(es)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	// Create two endpoints
	for _, ep := range []struct {
		route  string
		method string
	}{
		{"/ep1", "GET"},
		{"/ep2", "POST"},
	} {
		_, err := es.CreateEndpoint(context.Background(), CreateEndpointInput{
			OwnerID: ownerID, ProviderID: provider.ID,
			Route: ep.route, Method: ep.method,
			PriceAmount: numericFromInt64(10),
			Currency:    repository.CurrencyXLM,
			Status:      repository.EndpointStatusActive,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+provider.ID.String()+"/endpoints", nil)
	req.SetPathValue("providerId", provider.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      ownerID.String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.ListEndpoints(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var items []repository.ApiEndpoint
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(items))
	}
}

func TestListEndpointsHandler_Empty(t *testing.T) {
	eh, ownerID, provider := setupEndpointTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+provider.ID.String()+"/endpoints", nil)
	req.SetPathValue("providerId", provider.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      ownerID.String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.ListEndpoints(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var items []repository.ApiEndpoint
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if items == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 endpoints, got %d", len(items))
	}
}

func TestListEndpointsHandler_StatusFilter(t *testing.T) {
	mq := newMockQuerier()
	ps := NewProviderService(mq)
	es := NewEndpointService(mq)
	eh := NewEndpointHandler(es)
	ownerID := uuid.New()

	provider, err := ps.CreateProvider(context.Background(), ownerID, "Test", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	// Create one active and one draft endpoint
	es.CreateEndpoint(context.Background(), CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: "/active", Method: http.MethodGet,
		PriceAmount: numericFromInt64(10),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	es.CreateEndpoint(context.Background(), CreateEndpointInput{
		OwnerID: ownerID, ProviderID: provider.ID,
		Route: "/draft", Method: http.MethodGet,
		PriceAmount: numericFromInt64(20),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusDraft,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+provider.ID.String()+"/endpoints?status=active", nil)
	req.SetPathValue("providerId", provider.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      ownerID.String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.ListEndpoints(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var items []repository.ApiEndpoint
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(items))
	}
	if items[0].Route != "/active" {
		t.Errorf("expected route %q, got %q", "/active", items[0].Route)
	}
}

func TestListEndpointsHandler_InvalidStatusFilter(t *testing.T) {
	eh, ownerID, provider := setupEndpointTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+provider.ID.String()+"/endpoints?status=bogus", nil)
	req.SetPathValue("providerId", provider.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      ownerID.String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.ListEndpoints(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListEndpointsHandler_ProviderNotFound(t *testing.T) {
	eh := NewEndpointHandler(NewEndpointService(newMockQuerier()))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+uuid.New().String()+"/endpoints", nil)
	req.SetPathValue("providerId", uuid.New().String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.ListEndpoints(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListEndpointsHandler_NotOwner(t *testing.T) {
	eh, _, provider := setupEndpointTest(t)
	otherUser := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+provider.ID.String()+"/endpoints", nil)
	req.SetPathValue("providerId", provider.ID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      otherUser.String(),
			IsAuthenticated: true,
		}),
	)

	rec := httptest.NewRecorder()
	eh.ListEndpoints(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListEndpointsHandler_Unauthenticated(t *testing.T) {
	eh := NewEndpointHandler(NewEndpointService(newMockQuerier()))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/providers/"+uuid.New().String()+"/endpoints", nil)
	rec := httptest.NewRecorder()
	eh.ListEndpoints(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
