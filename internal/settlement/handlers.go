package settlement

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const errKey = "error"

const msgAuthRequired = "authentication required"

const maxLimit = 100

const defaultLimit = 20

const maxMonthlyHistoryMonths = 24

const cursorPartsCount = 2

type settlementBatchItem struct {
	ID          string                `json:"id"`
	Status      string                `json:"status"`
	TotalAmount string                `json:"total_amount"`
	Currency    string                `json:"currency"`
	EntryCount  int32                 `json:"entry_count"`
	TxHash      *string               `json:"tx_hash,omitempty"`
	CreatedAt   string                `json:"created_at"`
	CompletedAt *string               `json:"completed_at,omitempty"`
	Entries     []settlementEntryItem `json:"entries"`
}

type settlementEntryItem struct {
	ID            string `json:"id"`
	ProviderID    string `json:"provider_id"`
	ProviderName  string `json:"provider_name"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	WalletAddress string `json:"wallet_address"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

type settlementListResponse struct {
	Data       []settlementBatchItem `json:"data"`
	NextCursor *string               `json:"next_cursor"`
}

type monthlySettlement struct {
	Month  string `json:"month"`
	Amount string `json:"amount"`
}

type settlementSummaryResponse struct {
	TotalSettled   string              `json:"total_settled"`
	Currency       string              `json:"currency"`
	MonthlyHistory []monthlySettlement `json:"monthly_history"`
}

type settlementThresholdResponse struct {
	MinThreshold string `json:"min_threshold"`
	Currency     string `json:"currency"`
}

type OwnerLister interface {
	GetSettlementHistoryByOwner(ctx context.Context, ownerID uuid.UUID, cursorTs time.Time, cursorID uuid.UUID, limit int32, status string) ([]repository.SettlementBatch, map[uuid.UUID][]repository.ListSettlementEntriesByBatchWithProviderRow, error)
}

type OwnerSummarizer interface {
	GetSettlementSummaryByOwner(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error)
	GetSettlementMonthlyHistoryByOwner(ctx context.Context, ownerID uuid.UUID, limit int32) ([]MonthlySettlement, error)
}

type MonthlySettlement struct {
	Month  time.Time
	Amount decimal.Decimal
}

type Handler struct {
	lister     OwnerLister
	summarizer OwnerSummarizer
	threshold  decimal.Decimal
}

func NewHandler(lister OwnerLister, summarizer OwnerSummarizer, threshold decimal.Decimal) *Handler {
	return &Handler{
		lister:     lister,
		summarizer: summarizer,
		threshold:  threshold,
	}
}

func (h *Handler) ListSettlements(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})

		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})

		return
	}

	limit := int32(defaultLimit)
	if l := r.URL.Query().Get("limit"); l != "" {
		v, err := strconv.ParseInt(l, 10, 32)
		if err != nil || v < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid limit"})

			return
		}
		limit = int32(v)
		if limit > maxLimit {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "limit exceeds maximum"})

			return
		}
	}

	cursorTs, cursorID := parseCursor(r.URL.Query().Get("cursor"))

	status := r.URL.Query().Get("status")

	batches, entriesMap, err := h.lister.GetSettlementHistoryByOwner(r.Context(), ownerID, cursorTs, cursorID, limit+1, status)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to retrieve settlement history"})

		return
	}

	count := len(batches)
	hasNext := count > int(limit)
	if hasNext {
		count = int(limit)
	}

	items := make([]settlementBatchItem, 0, count)
	for i := range count {
		batch := batches[i]

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
		entryItems := make([]settlementEntryItem, 0, len(entries))

		for _, entry := range entries {
			amount := "0"
			if d, err := NumericToDecimal(entry.Amount); err == nil {
				amount = d.String()
			}

			entryItems = append(entryItems, settlementEntryItem{
				ID:            entry.ID.String(),
				ProviderID:    entry.ProviderID.String(),
				ProviderName:  entry.ProviderName,
				Amount:        amount,
				Currency:      string(entry.Currency),
				WalletAddress: entry.WalletAddress,
				Status:        string(entry.Status),
				CreatedAt:     entry.CreatedAt.UTC().Format(time.RFC3339),
			})
		}

		var txHash *string
		if batch.TxHash.Valid {
			s := batch.TxHash.String
			txHash = &s
		}

		items = append(items, settlementBatchItem{
			ID:          batch.ID.String(),
			Status:      string(batch.Status),
			TotalAmount: totalAmount,
			Currency:    string(batch.Currency),
			EntryCount:  batch.EntryCount,
			TxHash:      txHash,
			CreatedAt:   batch.CreatedAt.UTC().Format(time.RFC3339),
			CompletedAt: completedAt,
			Entries:     entryItems,
		})
	}

	var nextCursor *string
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		cursor := last.CreatedAt + "|" + last.ID
		nextCursor = &cursor
	}

	writeJSON(w, http.StatusOK, settlementListResponse{
		Data:       items,
		NextCursor: nextCursor,
	})
}

func (h *Handler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	consumer := gatewaycontext.GetConsumerInfo(r.Context())
	if !consumer.IsAuthenticated || consumer.ConsumerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{errKey: msgAuthRequired})

		return
	}

	ownerID, err := uuid.Parse(consumer.ConsumerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "invalid consumer identity"})

		return
	}

	totalSettled, err := h.summarizer.GetSettlementSummaryByOwner(r.Context(), ownerID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to retrieve settlement summary"})

		return
	}

	monthly, err := h.summarizer.GetSettlementMonthlyHistoryByOwner(r.Context(), ownerID, maxMonthlyHistoryMonths)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to retrieve settlement history"})

		return
	}

	monthlyItems := make([]monthlySettlement, 0, len(monthly))
	for _, m := range monthly {
		monthlyItems = append(monthlyItems, monthlySettlement{
			Month:  m.Month.UTC().Format("2006-01"),
			Amount: m.Amount.String(),
		})
	}

	writeJSON(w, http.StatusOK, settlementSummaryResponse{
		TotalSettled:   totalSettled.String(),
		Currency:       "XLM",
		MonthlyHistory: monthlyItems,
	})
}

func (h *Handler) HandleThreshold(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, settlementThresholdResponse{
		MinThreshold: h.threshold.String(),
		Currency:     "XLM",
	})
}

func parseCursor(s string) (time.Time, uuid.UUID) {
	if s == "" {
		return time.Time{}, uuid.UUID{}
	}
	parts := strings.SplitN(s, "|", cursorPartsCount)
	if len(parts) != cursorPartsCount {
		return time.Time{}, uuid.UUID{}
	}
	t, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return time.Time{}, uuid.UUID{}
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.UUID{}
	}
	return t, id
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson
}
