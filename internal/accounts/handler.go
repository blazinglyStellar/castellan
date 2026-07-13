package accounts

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stellar/go-stellar-sdk/keypair"
)

const errKey = "error"

const (
	defaultLimit = 50
	maxLimit     = 100
)

type Handler struct {
	service    *Service
	horizonURL string
}

func NewHandler(service *Service, horizonURL string) *Handler {
	return &Handler{service: service, horizonURL: horizonURL}
}

func (h *Handler) resolveOwnerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: "authentication required"})
		return uuid.UUID{}, false
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})
		return uuid.UUID{}, false
	}

	return ownerID, true
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.resolveOwnerID(w, r)
	if !ok {
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

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.resolveOwnerID(w, r)
	if !ok {
		return
	}

	balance, err := h.service.GetBalance(r.Context(), ownerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to get balance"})
		return
	}

	if balance == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{errKey: "account not found"})
		return
	}

	writeJSON(w, http.StatusOK, balance)
}

func (h *Handler) ListEntries(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.resolveOwnerID(w, r)
	if !ok {
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
	ownerID, ok := h.resolveOwnerID(w, r)
	if !ok {
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

type updatePayoutAddressRequest struct {
	PayoutStellarAddress string `json:"payout_stellar_address"`
}

func (h *Handler) CheckPayoutAddress(w http.ResponseWriter, r *http.Request) {
	addr := strings.TrimSpace(r.URL.Query().Get("address"))
	if addr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "address query parameter is required"})
		return
	}

	if _, err := keypair.Parse(addr); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"valid": false, "message": "invalid stellar address format"})
		return
	}

	client := horizonclient.Client{
		HorizonURL: h.horizonURL,
		HTTP:       horizonclient.DefaultTestNetClient.HTTP,
	}
	if _, err := client.AccountDetail(horizonclient.AccountRequest{AccountID: addr}); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"valid": false, "message": "address not found on the stellar network"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"valid": true})
}

func (h *Handler) UpdatePayoutAddress(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := h.resolveOwnerID(w, r)
	if !ok {
		return
	}

	var req updatePayoutAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid request body"})
		return
	}

	addr := strings.TrimSpace(req.PayoutStellarAddress)
	if addr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "payout_stellar_address is required"})
		return
	}

	if _, err := keypair.Parse(addr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid stellar address"})
		return
	}

	user, err := h.service.SetPayoutAddress(r.Context(), ownerID, addr)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to save payout address"})
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
