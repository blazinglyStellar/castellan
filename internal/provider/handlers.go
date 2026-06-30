package provider

import (
	"encoding/json"
	"net/http"

	"castellan/internal/repository/db"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
)

type createProviderRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

type Handler struct {
	service *ProviderService
}

func NewProviderHandler(service *ProviderService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid consumer identity"})
		return
	}

	var req createProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	provider, err := h.service.CreateProvider(r.Context(), ownerID, req.Name, req.BaseURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, provider)
}

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid consumer identity"})
		return
	}

	providers, err := h.service.ListProviders(r.Context(), ownerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list providers"})
		return
	}

	if providers == nil {
		providers = []repository.Provider{}
	}

	writeJSON(w, http.StatusOK, providers)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
