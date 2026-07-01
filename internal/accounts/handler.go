package accounts

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	gatewaycontext "castellan/internal/gateway/context"
)

const errKey = "error"

const (
	defaultLimit = 50
	maxLimit     = 100
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
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

	account, err := h.service.GetAccount(r.Context(), ownerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to get account"})
		return
	}

	if account == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "account not found"})
		return
	}

	writeJSON(w, http.StatusOK, account)
}

func (h *Handler) ListEntries(w http.ResponseWriter, r *http.Request) {
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

	account, err := h.service.GetAccount(r.Context(), ownerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to get account"})
		return
	}

	if account == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "account not found"})
		return
	}

	limit := int32(defaultLimit)
	if l := r.URL.Query().Get("limit"); l != "" {
		parsed, err := strconv.ParseInt(l, 10, 32)
		if err != nil || parsed < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid limit"})
			return
		}
		limit = int32(parsed)
		if limit > maxLimit {
			limit = maxLimit
		}
	}

	offset := int32(0)
	if o := r.URL.Query().Get("offset"); o != "" {
		parsed, err := strconv.ParseInt(o, 10, 32)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid offset"})
			return
		}
		offset = int32(parsed)
	}

	entryType := r.URL.Query().Get("type")

	result, err := h.service.ListEntries(r.Context(), account.ID, entryType, limit, offset)
	if err != nil {
		if errors.Is(err, errInvalidEntryType) {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to list entries"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetEntry(w http.ResponseWriter, r *http.Request) {
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

	entryID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid entry id"})
		return
	}

	entry, err := h.service.GetEntry(r.Context(), entryID, ownerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to get entry"})
		return
	}

	if entry == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "entry not found"})
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
