package accounts

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"castellan/internal/repository/db"
)

var errInvalidEntryType = errors.New("invalid entry type")

var validEntryTypes = map[string]repository.EntryType{
	"deposit":     repository.EntryTypeDeposit,
	"reservation": repository.EntryTypeReservation,
	"deduction":   repository.EntryTypeDeduction,
	"refund":      repository.EntryTypeRefund,
	"settlement":  repository.EntryTypeSettlement,
}

type Service struct {
	queries repository.Querier
}

func NewService(queries repository.Querier) *Service {
	return &Service{queries: queries}
}

type AccountResponse struct {
	ID        uuid.UUID `json:"id"`
	Balance   string    `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

type EntryResponse struct {
	ID            uuid.UUID `json:"id"`
	EntryType     string    `json:"entry_type"`
	Amount        string    `json:"amount"`
	BalanceAfter  string    `json:"balance_after"`
	Currency      string    `json:"currency"`
	ReferenceType *string   `json:"reference_type,omitempty"`
	Status        string    `json:"status"`
	Description   *string   `json:"description,omitempty"`
	CreatedAt     string    `json:"created_at"`
}

type ListEntriesResponse struct {
	Entries []EntryResponse `json:"entries"`
	Total   int64           `json:"total"`
	Limit   int32           `json:"limit"`
	Offset  int32           `json:"offset"`
}

func (s *Service) GetAccount(ctx context.Context, ownerID uuid.UUID) (*AccountResponse, error) {
	user, err := s.queries.GetUserByID(ctx, ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	balance, err := numericToDecimal(user.Balance)
	if err != nil {
		return nil, fmt.Errorf("convert balance: %w", err)
	}

	return &AccountResponse{
		ID:        user.ID,
		Balance:   balance.String(),
		Currency:  string(user.Currency),
		CreatedAt: user.CreatedAt.Format(isoFormat),
		UpdatedAt: user.AccountUpdatedAt.Format(isoFormat),
	}, nil
}

func (s *Service) ListEntries(ctx context.Context, userID uuid.UUID, entryType string, limit, offset int32) (*ListEntriesResponse, error) {
	var entries []repository.LedgerEntry
	var total int64
	var err error

	if entryType != "" {
		et, ok := validEntryTypes[entryType]
		if !ok {
			return nil, fmt.Errorf("%w: %s", errInvalidEntryType, entryType)
		}

		entries, err = s.queries.ListLedgerEntriesByAccountAndType(ctx, repository.ListLedgerEntriesByAccountAndTypeParams{
			UserID:    userID,
			EntryType: et,
			Limit:     limit,
			Offset:    offset,
		})
		if err != nil {
			return nil, fmt.Errorf("list entries by type: %w", err)
		}

		total, err = s.queries.CountLedgerEntriesByAccountAndType(ctx, repository.CountLedgerEntriesByAccountAndTypeParams{
			UserID:    userID,
			EntryType: et,
		})
		if err != nil {
			return nil, fmt.Errorf("count entries by type: %w", err)
		}
	} else {
		entries, err = s.queries.ListLedgerEntriesByAccount(ctx, repository.ListLedgerEntriesByAccountParams{
			UserID: userID,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("list entries: %w", err)
		}

		total, err = s.queries.CountLedgerEntriesByAccount(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("count entries: %w", err)
		}
	}

	resp := &ListEntriesResponse{
		Entries: make([]EntryResponse, 0, len(entries)),
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}

	for _, e := range entries {
		entry, err := toEntryResponse(e)
		if err != nil {
			return nil, err
		}
		resp.Entries = append(resp.Entries, *entry)
	}

	return resp, nil
}

type BalanceResponse struct {
	Balance          string `json:"balance"`
	Currency         string `json:"currency"`
	AvailableBalance string `json:"available_balance"`
}

func (s *Service) GetBalance(ctx context.Context, ownerID uuid.UUID) (*BalanceResponse, error) {
	user, err := s.queries.GetUserByID(ctx, ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	balance, err := numericToDecimal(user.Balance)
	if err != nil {
		return nil, fmt.Errorf("convert balance: %w", err)
	}

	reservations, err := s.queries.GetActiveReservationsSum(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get active reservations: %w", err)
	}
	reservationsDec, err := numericToDecimal(reservations)
	if err != nil {
		return nil, fmt.Errorf("convert reservations: %w", err)
	}

	available := balance.Sub(reservationsDec)

	return &BalanceResponse{
		Balance:          balance.String(),
		Currency:         string(user.Currency),
		AvailableBalance: available.String(),
	}, nil
}

func (s *Service) GetEntry(ctx context.Context, entryID, ownerID uuid.UUID) (*EntryResponse, error) {
	entry, err := s.queries.GetLedgerEntryByIDAndOwner(ctx, repository.GetLedgerEntryByIDAndOwnerParams{
		ID:     entryID,
		UserID: ownerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get entry: %w", err)
	}

	return toEntryResponse(entry)
}

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, nil
	}
	if n.NaN {
		return decimal.Zero, errors.New("convert numeric: value is NaN")
	}
	return decimal.NewFromBigInt(n.Int, n.Exp), nil
}

const isoFormat = "2006-01-02T15:04:05Z07:00"

func toEntryResponse(e repository.LedgerEntry) (*EntryResponse, error) {
	amount, err := numericToDecimal(e.Amount)
	if err != nil {
		return nil, fmt.Errorf("convert amount: %w", err)
	}

	balanceAfter, err := numericToDecimal(e.BalanceAfter)
	if err != nil {
		return nil, fmt.Errorf("convert balance_after: %w", err)
	}

	var refType *string
	if e.ReferenceType.Valid {
		refType = &e.ReferenceType.String
	}

	var desc *string
	if e.Description.Valid {
		desc = &e.Description.String
	}

	return &EntryResponse{
		ID:            e.ID,
		EntryType:     string(e.EntryType),
		Amount:        amount.String(),
		BalanceAfter:  balanceAfter.String(),
		Currency:      string(e.Currency),
		ReferenceType: refType,
		Status:        string(e.Status),
		Description:   desc,
		CreatedAt:     e.CreatedAt.Format(isoFormat),
	}, nil
}
