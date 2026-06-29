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
	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func authenticatedRequest(t *testing.T, method, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if method == http.MethodGet {
		req = httptest.NewRequest(method, "/api/v1/keys", nil)
	} else {
		req = httptest.NewRequest(method, "/api/v1/keys", strings.NewReader(body))
	}
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)

	return req
}

func authenticatedRequestWithUserID(t *testing.T, userID uuid.UUID) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys", nil)
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      userID.String(),
			IsAuthenticated: true,
		}),
	)

	return req
}

func TestCreateKey_Success(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	h.CreateKey(rec, authenticatedRequest(t, http.MethodPost, `{"label":"Production key"}`))

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
	h.CreateKey(rec, authenticatedRequest(t, http.MethodPost, `{"label":"test"}`))

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
				req = authenticatedRequest(t, http.MethodPost, tt.body)
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

func TestListKeys_Success(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			userID: {
				{
					ID:        uuid.New(),
					UserID:    userID,
					KeyHash:   "secret",
					Label:     pgtype.Text{String: "Production", Valid: true},
					Status:    repository.ApiKeyStatusActive,
					CreatedAt: now,
				},
			},
		},
	}
	h := NewKeyHandler(NewKeyService(mq))
	rec := httptest.NewRecorder()
	h.ListKeys(rec, authenticatedRequestWithUserID(t, userID))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var items []ListKeysItem
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 key, got %d", len(items))
	}
	if items[0].Label.String != "Production" {
		t.Errorf("expected label 'Production', got %q", items[0].Label.String)
	}
}

func TestListKeys_Empty(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	h.ListKeys(rec, authenticatedRequest(t, http.MethodGet, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var items []ListKeysItem
	if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if items == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(items) != 0 {
		t.Errorf("expected 0 keys, got %d", len(items))
	}
}

func TestListKeys_Unauthenticated(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/keys", nil)
	h.ListKeys(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func revokeRequest(t *testing.T, keyID uuid.UUID, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/"+keyID.String()+"/revoke", strings.NewReader(body))
	req.SetPathValue("id", keyID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)
	return req
}

func TestRevokeKeyHandler_Success(t *testing.T) {
	userID := uuid.New()
	keyID := uuid.New()
	now := time.Now().UTC()
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			userID: {
				{
					ID:        keyID,
					UserID:    userID,
					KeyHash:   "hash",
					Label:     pgtype.Text{String: "Production key", Valid: true},
					Status:    repository.ApiKeyStatusActive,
					CreatedAt: now,
				},
			},
		},
	}
	h := NewKeyHandler(NewKeyService(mq))
	rec := httptest.NewRecorder()
	req := revokeRequest(t, keyID, "")
	// Override consumer to match the key's user
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      userID.String(),
			IsAuthenticated: true,
		}),
	)
	h.RevokeKey(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp revokeKeyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "revoked" {
		t.Errorf("expected status 'revoked', got %q", resp.Status)
	}
	if resp.Label != "Production key" {
		t.Errorf("expected label 'Production key', got %q", resp.Label)
	}
}

func TestRevokeKeyHandler_AlreadyRevoked(t *testing.T) {
	userID := uuid.New()
	keyID := uuid.New()
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			userID: {
				{
					ID:     keyID,
					UserID: userID,
					Status: repository.ApiKeyStatusRevoked,
				},
			},
		},
	}
	h := NewKeyHandler(NewKeyService(mq))
	rec := httptest.NewRecorder()
	req := authenticatedRequestWithUserID(t, userID)
	req.SetPathValue("id", keyID.String())
	h.RevokeKey(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestRevokeKeyHandler_NotFound(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	req := authenticatedRequestWithUserID(t, uuid.New())
	req.SetPathValue("id", uuid.New().String())
	h.RevokeKey(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestRevokeKeyHandler_Unauthenticated(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/some-id/revoke", nil)
	req.SetPathValue("id", "some-id")
	h.RevokeKey(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestRotateKeyHandler_Success(t *testing.T) {
	userID := uuid.New()
	keyID := uuid.New()
	now := time.Now().UTC()
	mq := &mockQuerier{
		keysByUser: map[uuid.UUID][]repository.ApiKey{
			userID: {
				{
					ID:        keyID,
					UserID:    userID,
					KeyHash:   "old-hash",
					Label:     pgtype.Text{String: "Production key", Valid: true},
					Status:    repository.ApiKeyStatusActive,
					CreatedAt: now,
				},
			},
		},
	}
	h := NewKeyHandler(NewKeyService(mq))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/"+keyID.String()+"/rotate", nil)
	req.SetPathValue("id", keyID.String())
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      userID.String(),
			IsAuthenticated: true,
		}),
	)
	h.RotateKey(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp createKeyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.HasPrefix(resp.Key, "ca_") {
		t.Errorf("expected key to start with ca_, got %q", resp.Key)
	}
	if resp.Status != "active" {
		t.Errorf("expected status 'active', got %q", resp.Status)
	}
	if resp.Label != "Production key" {
		t.Errorf("expected label 'Production key', got %q", resp.Label)
	}
}

func TestRotateKeyHandler_NotFound(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	req := authenticatedRequestWithUserID(t, uuid.New())
	req.SetPathValue("id", uuid.New().String())
	h.RotateKey(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestRotateKeyHandler_Unauthenticated(t *testing.T) {
	h := NewKeyHandler(NewKeyService(&mockQuerier{}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/keys/some-id/rotate", nil)
	req.SetPathValue("id", "some-id")
	h.RotateKey(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
