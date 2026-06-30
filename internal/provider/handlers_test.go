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
