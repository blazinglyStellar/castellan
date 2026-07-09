package auth

import (
	"encoding/json"
	"net/http"
	"time"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const errKey = "error"

type createKeyRequest struct {
	Label     string     `json:"label"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type createKeyResponse struct {
	Key       string     `json:"key"`
	ID        uuid.UUID  `json:"id"`
	Label     string     `json:"label"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type updateKeyRequest struct {
	Label     *string    `json:"label,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type updateKeyResponse struct {
	ID        uuid.UUID  `json:"id"`
	Label     string     `json:"label"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type revokeKeyResponse struct {
	ID        uuid.UUID  `json:"id"`
	Label     string     `json:"label"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type ListKeysItem struct {
	ID        uuid.UUID               `json:"id"`
	Label     pgtype.Text             `json:"label"`
	Status    repository.ApiKeyStatus `json:"status"`
	CreatedAt time.Time               `json:"created_at"`
	ExpiresAt pgtype.Timestamptz      `json:"expires_at"`
}

type KeyHandler struct {
	service *KeyService
}

func NewKeyHandler(service *KeyService) *KeyHandler {
	return &KeyHandler{service: service}
}

func (h *KeyHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	userID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	var req createKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid request body"})
		return
	}

	if req.Label == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "label is required"})
		return
	}

	if req.ExpiresAt == nil {
		defaultExpiry := time.Now().UTC().Add(30 * 24 * time.Hour)
		req.ExpiresAt = &defaultExpiry
	}

	if req.ExpiresAt.Before(time.Now()) {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "expires_at must be in the future"})
		return
	}

	rawKey, apiKey, err := h.service.GenerateKey(r.Context(), userID, req.Label, req.ExpiresAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to create API key"})
		return
	}

	resp := createKeyResponse{
		Key:       rawKey,
		ID:        apiKey.ID,
		Label:     apiKey.Label.String,
		Status:    string(apiKey.Status),
		CreatedAt: apiKey.CreatedAt,
	}
	if apiKey.ExpiresAt.Valid {
		resp.ExpiresAt = &apiKey.ExpiresAt.Time
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *KeyHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	userID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	keys, err := h.service.ListKeys(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to list API keys"})
		return
	}

	writeJSON(w, http.StatusOK, keys)
}

func (h *KeyHandler) UpdateKey(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	userID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	keyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid key id"})
		return
	}

	var req updateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid request body"})
		return
	}

	var key repository.ApiKey

	if req.Label != nil {
		key, err = h.service.UpdateKeyLabel(r.Context(), keyID, userID, *req.Label)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{errKey: "key not found"})
			return
		}
	}

	if req.ExpiresAt != nil {
		key, err = h.service.UpdateKeyExpiration(r.Context(), keyID, userID, req.ExpiresAt)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{errKey: "key not found"})
			return
		}
	}

	if req.Label == nil && req.ExpiresAt == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "nothing to update"})
		return
	}

	resp := updateKeyResponse{
		ID:        key.ID,
		Label:     key.Label.String,
		Status:    string(key.Status),
		CreatedAt: key.CreatedAt,
	}
	if key.ExpiresAt.Valid {
		resp.ExpiresAt = &key.ExpiresAt.Time
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *KeyHandler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	userID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	keyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid key id"})
		return
	}

	key, err := h.service.RevokeKey(r.Context(), keyID, userID)
	if err != nil {
		if err.Error() == "key already revoked" {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "key already revoked"})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "key not found"})
		return
	}

	resp := revokeKeyResponse{
		ID:        key.ID,
		Label:     key.Label.String,
		Status:    string(key.Status),
		CreatedAt: key.CreatedAt,
	}
	if key.ExpiresAt.Valid {
		resp.ExpiresAt = &key.ExpiresAt.Time
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *KeyHandler) RotateKey(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	userID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	keyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid key id"})
		return
	}

	rawKey, newKey, err := h.service.RotateKey(r.Context(), keyID, userID)
	if err != nil {
		if err.Error() == "key is not active" {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "key is not active"})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "key not found"})
		return
	}

	resp := createKeyResponse{
		Key:       rawKey,
		ID:        newKey.ID,
		Label:     newKey.Label.String,
		Status:    string(newKey.Status),
		CreatedAt: newKey.CreatedAt,
	}
	if newKey.ExpiresAt.Valid {
		resp.ExpiresAt = &newKey.ExpiresAt.Time
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
