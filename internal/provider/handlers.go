package provider

import (
	"encoding/json"
	"net/http"

	"castellan/internal/repository/db"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
)

const errKey = "error"

type createProviderRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

type updateProviderRequest struct {
	Name    *string `json:"name,omitempty"`
	BaseURL *string `json:"base_url,omitempty"`
}

type updateProviderStatusRequest struct {
	Status repository.ProviderStatus `json:"status"`
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
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	var req createProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid request body"})
		return
	}

	provider, err := h.service.CreateProvider(r.Context(), ownerID, req.Name, req.BaseURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, provider)
}

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	providers, err := h.service.ListProviders(r.Context(), ownerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to list providers"})
		return
	}

	if providers == nil {
		providers = []repository.Provider{}
	}

	writeJSON(w, http.StatusOK, providers)
}

func (h *Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	providerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid provider id"})
		return
	}

	provider, err := h.service.GetProviderByID(r.Context(), providerID, ownerID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "provider not found"})
		return
	}

	writeJSON(w, http.StatusOK, provider)
}

func (h *Handler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	providerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid provider id"})
		return
	}

	var req updateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid request body"})
		return
	}

	provider, err := h.service.PartialUpdateProvider(r.Context(), providerID, ownerID, req.Name, req.BaseURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, provider)
}

func (h *Handler) UpdateProviderStatus(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	providerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid provider id"})
		return
	}

	var req updateProviderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid request body"})
		return
	}

	provider, err := h.service.UpdateProviderStatus(r.Context(), providerID, ownerID, req.Status)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, provider)
}

func (h *Handler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return
	}

	providerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid provider id"})
		return
	}

	if _, err := h.service.DeleteProvider(r.Context(), providerID, ownerID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "provider not found"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
