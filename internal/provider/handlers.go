package provider

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"castellan/internal/repository/db"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

const errKey = "error"

const msgAuthRequired = "authentication required"

const msgInvalidConsumerID = "invalid consumer identity"

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
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})
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
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})
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

//nolint:dupl
func (h *Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})
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
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})
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

//nolint:dupl
func (h *Handler) UpdateProviderStatus(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})
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
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})
		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})
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

type createEndpointRequest struct {
	Route       string          `json:"route"`
	Method      string          `json:"method"`
	PriceAmount decimal.Decimal `json:"price_amount"`
	Currency    string          `json:"currency,omitempty"`
	RateLimit   *int32          `json:"rate_limit,omitempty"`
}

type updateEndpointRequest struct {
	Route       *string          `json:"route,omitempty"`
	Method      *string          `json:"method,omitempty"`
	PriceAmount *decimal.Decimal `json:"price_amount,omitempty"`
	Currency    *string          `json:"currency,omitempty"`
	RateLimit   *int32           `json:"rate_limit,omitempty"`
}

type updateEndpointStatusRequest struct {
	Status repository.EndpointStatus `json:"status"`
}

type EndpointHandler struct {
	service *EndpointService
}

func NewEndpointHandler(service *EndpointService) *EndpointHandler {
	return &EndpointHandler{service: service}
}

func (h *EndpointHandler) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})

		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})

		return
	}

	providerID, err := uuid.Parse(r.PathValue("providerId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid provider id"})

		return
	}

	var req createEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid request body"})

		return
	}

	currency := repository.CurrencyXLM
	if req.Currency != "" {
		currency = repository.Currency(req.Currency)
	}

	var rateLimit pgtype.Int4
	if req.RateLimit != nil {
		rateLimit = pgtype.Int4{Int32: *req.RateLimit, Valid: true}
	}

	priceNum := pgtype.Numeric{}
	if err := priceNum.Scan(req.PriceAmount.String()); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid price_amount"})

		return
	}

	input := CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  providerID,
		Route:       req.Route,
		Method:      strings.ToUpper(req.Method),
		PriceAmount: priceNum,
		Currency:    currency,
		RateLimit:   rateLimit,
		Status:      repository.EndpointStatusDraft,
	}

	endpoint, err := h.service.CreateEndpoint(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicateEndpoint):
			writeJSON(w, http.StatusConflict, map[string]string{errKey: err.Error()})
		case errors.Is(err, ErrEndpointNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{errKey: "provider not found"})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: err.Error()})
		}

		return
	}

	writeJSON(w, http.StatusCreated, endpoint)
}

func (h *EndpointHandler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})

		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})

		return
	}

	providerID, err := uuid.Parse(r.PathValue("providerId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid provider id"})

		return
	}

	var statusFilter *repository.EndpointStatus
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		s := repository.EndpointStatus(statusParam)
		if err := validateEndpointStatus(s); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: err.Error()})

			return
		}
		statusFilter = &s
	}

	endpoints, err := h.service.ListEndpoints(r.Context(), providerID, ownerID, statusFilter)
	if err != nil {
		if errors.Is(err, ErrEndpointNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{errKey: "provider not found"})

			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: err.Error()})

		return
	}

	if endpoints == nil {
		endpoints = []repository.ApiEndpoint{}
	}

	writeJSON(w, http.StatusOK, endpoints)
}

//nolint:dupl
func (h *EndpointHandler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})

		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})

		return
	}

	endpointID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid endpoint id"})

		return
	}

	endpoint, err := h.service.GetEndpointByID(r.Context(), endpointID, ownerID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "endpoint not found"})

		return
	}

	writeJSON(w, http.StatusOK, endpoint)
}

func (h *EndpointHandler) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})

		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})

		return
	}

	endpointID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid endpoint id"})

		return
	}

	var req updateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid request body"})

		return
	}

	var partialInput PartialUpdateEndpointInput
	if req.Route != nil {
		partialInput.Route = req.Route
	}
	if req.Method != nil {
		method := strings.ToUpper(*req.Method)
		partialInput.Method = &method
	}
	if req.PriceAmount != nil {
		var priceNum pgtype.Numeric
		if err := priceNum.Scan(req.PriceAmount.String()); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid price_amount"})

			return
		}
		partialInput.PriceAmount = &priceNum
	}
	if req.Currency != nil {
		currency := repository.Currency(*req.Currency)
		partialInput.Currency = &currency
	}
	if req.RateLimit != nil {
		rateLimit := pgtype.Int4{Int32: *req.RateLimit, Valid: true}
		partialInput.RateLimit = &rateLimit
	}

	endpoint, err := h.service.PartialUpdateEndpoint(r.Context(), endpointID, ownerID, partialInput)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicateEndpoint):
			writeJSON(w, http.StatusConflict, map[string]string{errKey: err.Error()})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: err.Error()})
		}

		return
	}

	writeJSON(w, http.StatusOK, endpoint)
}

//nolint:dupl
func (h *EndpointHandler) UpdateEndpointStatus(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})

		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})

		return
	}

	endpointID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid endpoint id"})

		return
	}

	var req updateEndpointStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid request body"})

		return
	}

	endpoint, err := h.service.UpdateEndpointStatus(r.Context(), endpointID, ownerID, req.Status)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: err.Error()})

		return
	}

	writeJSON(w, http.StatusOK, endpoint)
}

func (h *EndpointHandler) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})

		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: msgInvalidConsumerID})

		return
	}

	endpointID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid endpoint id"})

		return
	}

	if _, err := h.service.DeleteEndpoint(r.Context(), endpointID, ownerID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "endpoint not found"})

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
