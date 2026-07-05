package settlement

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
)

const errKey = "error"

const msgAuthRequired = "authentication required"

const maxLimit = 100

type batchResponse struct {
	ID          string          `json:"id"`
	Status      string          `json:"status"`
	TotalAmount string          `json:"total_amount"`
	Currency    string          `json:"currency"`
	EntryCount  int32           `json:"entry_count"`
	CreatedAt   string          `json:"created_at"`
	CompletedAt *string         `json:"completed_at,omitempty"`
	Entries     []entryResponse `json:"entries"`
}

type entryResponse struct {
	ID            string `json:"id"`
	ProviderID    string `json:"provider_id"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	WalletAddress string `json:"wallet_address"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

type listSettlementsResponse struct {
	Batches []batchResponse `json:"batches"`
	Total   int64           `json:"total"`
}

type Lister interface {
	GetSettlementHistory(ctx context.Context, limit, offset int32) ([]repository.SettlementBatch, map[uuid.UUID][]repository.SettlementEntry, error)
	CountSettlementBatches(ctx context.Context) (int64, error)
}

type Handler struct {
	lister Lister
}

func NewHandler(lister Lister) *Handler {
	return &Handler{lister: lister}
}

func (h *Handler) ListSettlements(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})

		return
	}

	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err != nil || v < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid limit"})

			return
		}
		if v > maxLimit {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "limit exceeds maximum of 100"})

			return
		}
		limit = v
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		v, err := strconv.Atoi(o)
		if err != nil || v < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid offset"})

			return
		}
		if v > math.MaxInt32 {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "offset too large"})

			return
		}
		offset = v
	}

	batches, entriesMap, err := h.lister.GetSettlementHistory(r.Context(), int32(limit), int32(offset))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to retrieve settlement history"})

		return
	}

	total, err := h.lister.CountSettlementBatches(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to count settlement batches"})

		return
	}

	resp := listSettlementsResponse{
		Batches: make([]batchResponse, 0, len(batches)),
		Total:   total,
	}

	for _, batch := range batches {
		totalAmount := "0"
		if d, err := NumericToDecimal(batch.TotalAmount); err == nil {
			totalAmount = d.String()
		}

		var completedAt *string
		if batch.CompletedAt.Valid {
			s := batch.CompletedAt.Time.UTC().Format(time.RFC3339)
			completedAt = &s
		}

		entries := entriesMap[batch.ID]
		entryResponses := make([]entryResponse, 0, len(entries))

		for _, entry := range entries {
			amount := "0"
			if d, err := NumericToDecimal(entry.Amount); err == nil {
				amount = d.String()
			}

			entryResponses = append(entryResponses, entryResponse{
				ID:            entry.ID.String(),
				ProviderID:    entry.ProviderID.String(),
				Amount:        amount,
				Currency:      string(entry.Currency),
				WalletAddress: entry.WalletAddress,
				Status:        string(entry.Status),
				CreatedAt:     entry.CreatedAt.UTC().Format(time.RFC3339),
			})
		}

		resp.Batches = append(resp.Batches, batchResponse{
			ID:          batch.ID.String(),
			Status:      string(batch.Status),
			TotalAmount: totalAmount,
			Currency:    string(batch.Currency),
			EntryCount:  batch.EntryCount,
			CreatedAt:   batch.CreatedAt.UTC().Format(time.RFC3339),
			CompletedAt: completedAt,
			Entries:     entryResponses,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
