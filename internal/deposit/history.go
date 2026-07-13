package deposit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"castellan/internal/repository/db"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	defaultLimit     = 50
	maxLimit         = 100
	cursorPartsCount = 2
)

// DepositItem is the JSON representation of a single deposit in the list.
type DepositItem struct { //nolint:revive
	ID          string  `json:"id"`
	Amount      string  `json:"amount"`
	Currency    string  `json:"currency"`
	Memo        string  `json:"memo,omitempty"`
	TxHash      string  `json:"tx_hash"`
	Status      string  `json:"status"`
	FromAddress string  `json:"from_address"`
	CreatedAt   string  `json:"created_at"`
	ConfirmedAt *string `json:"confirmed_at,omitempty"`
}

// DepositListResponse is the cursor-paginated deposit list.
type DepositListResponse struct { //nolint:revive
	Data       []DepositItem `json:"data"`
	NextCursor *string       `json:"next_cursor"`
}

// ListDepositsCursor retrieves deposits with cursor pagination.
func (s *Service) ListDepositsCursor(ctx context.Context, userID uuid.UUID, cursorTs time.Time, cursorID uuid.UUID, limit int32) (*DepositListResponse, error) {
	_, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &DepositListResponse{Data: []DepositItem{}}, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	rows, err := s.queries.ListDepositsByAccountCursor(ctx, repository.ListDepositsByAccountCursorParams{
		UserID:  userID,
		Limit:   limit + 1,
		Column3: cursorTs,
		Column4: cursorID,
	})
	if err != nil {
		return nil, fmt.Errorf("list deposits: %w", err)
	}

	count := len(rows)
	hasNext := count > int(limit)
	if hasNext {
		count = int(limit)
	}

	items := make([]DepositItem, count)
	for i := range count {
		d := rows[i]
		amount, err := numericToDecimal(d.Amount)
		if err != nil {
			return nil, fmt.Errorf("convert deposit amount: %w", err)
		}

		var confirmedAt *string
		if d.ConfirmedAt.Valid {
			s := d.ConfirmedAt.Time.Format(time.RFC3339)
			confirmedAt = &s
		}

		memo := ""
		if d.Memo.Valid {
			memo = d.Memo.String
		}

		items[i] = DepositItem{
			ID:          d.ID.String(),
			Amount:      amount.String(),
			Currency:    string(d.Currency),
			Memo:        memo,
			TxHash:      d.TxHash,
			Status:      string(d.Status),
			FromAddress: d.FromAddress,
			CreatedAt:   d.CreatedAt.Format(time.RFC3339),
			ConfirmedAt: confirmedAt,
		}
	}

	var nextCursor *string
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		cursor := last.CreatedAt + "|" + last.ID
		nextCursor = &cursor
	}

	return &DepositListResponse{
		Data:       items,
		NextCursor: nextCursor,
	}, nil
}

func (h *Handler) ListDeposits(w http.ResponseWriter, r *http.Request) {
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

	limit := int32(defaultLimit)
	if l := r.URL.Query().Get("limit"); l != "" {
		v, err := strconv.ParseInt(l, 10, 32)
		if err != nil || v < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{errKey: "invalid limit"})

			return
		}
		limit = int32(v)
		if limit > maxLimit {
			limit = maxLimit
		}
	}

	cursorTs, cursorID := parseCursor(r.URL.Query().Get("cursor"))

	deposits, err := h.service.ListDepositsCursor(r.Context(), userID, cursorTs, cursorID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to list deposits"})

		return
	}

	writeJSON(w, http.StatusOK, deposits)
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
