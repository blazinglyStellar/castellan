package deposit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"castellan/internal/gateway/context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Response is the JSON representation of a single deposit.
type Response struct {
	ID          string  `json:"id"`
	Amount      string  `json:"amount"`
	Currency    string  `json:"currency"`
	TxHash      string  `json:"tx_hash"`
	Status      string  `json:"status"`
	FromAddress string  `json:"from_address"`
	CreatedAt   string  `json:"created_at"`
	ConfirmedAt *string `json:"confirmed_at,omitempty"`
}

// ListDeposits retrieves all deposits for the authenticated consumer's account.
func (s *Service) ListDeposits(ctx context.Context, userID uuid.UUID) ([]Response, error) {
	account, err := s.queries.GetAccountByOwnerID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []Response{}, nil
		}

		return nil, fmt.Errorf("get account: %w", err)
	}

	deposits, err := s.queries.ListDepositsByAccount(ctx, account.ID)
	if err != nil {
		return nil, fmt.Errorf("list deposits: %w", err)
	}

	resp := make([]Response, len(deposits))

	for i, d := range deposits {
		amount, err := numericToDecimal(d.Amount)
		if err != nil {
			return nil, fmt.Errorf("convert deposit amount: %w", err)
		}

		var confirmedAt *string
		if d.ConfirmedAt.Valid {
			s := d.ConfirmedAt.Time.Format(time.RFC3339)
			confirmedAt = &s
		}

		resp[i] = Response{
			ID:          d.ID.String(),
			Amount:      amount.String(),
			Currency:    string(d.Currency),
			TxHash:      d.TxHash,
			Status:      string(d.Status),
			FromAddress: d.FromAddress,
			CreatedAt:   d.CreatedAt.Format(time.RFC3339),
			ConfirmedAt: confirmedAt,
		}
	}

	return resp, nil
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

	deposits, err := h.service.ListDeposits(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{errKey: "failed to list deposits"})

		return
	}

	writeJSON(w, http.StatusOK, deposits)
}
