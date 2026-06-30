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
