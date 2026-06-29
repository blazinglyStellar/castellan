package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
)

func authenticatedRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys", strings.NewReader(body))
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)

	return req
}

func TestCreateKey_Success(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	h.CreateKey(rec, authenticatedRequest(t, `{"label":"Production key"}`))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp createKeyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !strings.HasPrefix(resp.Key, "ca_") {
		t.Errorf("expected key to start with ca_, got %q", resp.Key)
	}
	if resp.ID == uuid.Nil {
		t.Error("expected non-nil id")
	}
	if resp.Label != "Production key" {
		t.Errorf("expected label %q, got %q", "Production key", resp.Label)
	}
	if resp.Status != "active" {
		t.Errorf("expected status 'active', got %q", resp.Status)
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if resp.ExpiresAt == nil {
		t.Fatal("expected default expires_at (30 days)")
	}
	if resp.ExpiresAt.Before(time.Now()) {
		t.Error("expires_at should be in the future")
	}
}

func TestCreateKey_DefaultExpiryIs30Days(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	h.CreateKey(rec, authenticatedRequest(t, `{"label":"test"}`))

	var resp createKeyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedMin := time.Now().UTC().Add(29 * 24 * time.Hour)
	expectedMax := time.Now().UTC().Add(31 * 24 * time.Hour)

	if resp.ExpiresAt == nil {
		t.Fatal("expected expires_at to be set")
	}
	if resp.ExpiresAt.Before(expectedMin) || resp.ExpiresAt.After(expectedMax) {
		t.Errorf("expected expires_at ~30 days from now, got %v", resp.ExpiresAt)
	}
}

func TestCreateKey_ValidationErrors(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))

	tests := []struct {
		name       string
		body       string
		wantErr    string
		wantStatus int
	}{
		{
			name:       "missing label",
			body:       `{}`,
			wantErr:    "label is required",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "expires_at in the past",
			body:       `{"label":"test","expires_at":"2020-01-01T00:00:00Z"}`,
			wantErr:    "expires_at must be in the future",
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
				req = httptest.NewRequest(http.MethodPost, "/api/v1/keys", bytes.NewReader([]byte(tt.body)))
				req = req.WithContext(
					gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
						ConsumerID:      uuid.New().String(),
						IsAuthenticated: true,
					}),
				)
			} else {
				req = authenticatedRequest(t, tt.body)
			}

			h.CreateKey(rec, req)

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

func TestCreateKey_Unauthenticated(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys", strings.NewReader(`{"label":"test"}`))
	h.CreateKey(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
