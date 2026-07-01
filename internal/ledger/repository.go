package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"castellan/internal/repository/db"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	f64, err := n.Float64Value()
	if err != nil {
		return decimal.Zero, fmt.Errorf("convert numeric to float64: %w", err)
	}

	if !f64.Valid {
		return decimal.Zero, errors.New("numeric is null")
	}

	return decimal.NewFromFloat(f64.Float64), nil
}

func decimalToNumeric(d decimal.Decimal) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(d.String()); err != nil {
		return n, fmt.Errorf("scan decimal to numeric: %w", err)
	}

	return n, nil
}

func stringToUUID(s string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uuid: %w", err)
	}

	return parsed, nil
}

func pgtypeUUIDFromUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

type ReserveBalanceResult struct {
	Account repository.Account
	Entry   repository.LedgerEntry
}

func (r *Repository) ReserveBalance(
	ctx context.Context,
	ownerID uuid.UUID,
	amount decimal.Decimal,
	requestID string,
) (*ReserveBalanceResult, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive")
	}

	refUUID, err := stringToUUID(requestID)
	if err != nil {
		return nil, fmt.Errorf("invalid request id: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	q := repository.New(tx)

	if _, err := q.GetOrCreateAccount(ctx, ownerID); err != nil {
		return nil, fmt.Errorf("get or create account: %w", err)
	}

	account, err := q.GetAccountForUpdate(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("lock account: %w", err)
	}

	balanceDec, err := numericToDecimal(account.Balance)
	if err != nil {
		return nil, fmt.Errorf("parse balance: %w", err)
	}

	newBalance := balanceDec.Sub(amount)
	if newBalance.LessThan(decimal.Zero) {
		return nil, ErrInsufficientBalance
	}

	newBalanceNumeric, err := decimalToNumeric(newBalance)
	if err != nil {
		return nil, fmt.Errorf("convert new balance: %w", err)
	}

	account, err = q.UpdateAccountBalance(ctx, repository.UpdateAccountBalanceParams{
		ID:      account.ID,
		Balance: newBalanceNumeric,
	})
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	amountNumeric, err := decimalToNumeric(amount)
	if err != nil {
		return nil, fmt.Errorf("convert amount: %w", err)
	}

	entry, err := q.InsertLedgerEntry(ctx, repository.InsertLedgerEntryParams{
		AccountID:     account.ID,
		EntryType:     repository.EntryTypeReservation,
		Amount:        amountNumeric,
		BalanceAfter:  newBalanceNumeric,
		Currency:      repository.CurrencyXLM,
		ReferenceID:   pgtypeUUIDFromUUID(refUUID),
		ReferenceType: pgtype.Text{String: "gateway", Valid: true},
		Status:        repository.LedgerStatusPending,
		Description:   pgtype.Text{String: "reserved for API request", Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("insert ledger entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &ReserveBalanceResult{Account: account, Entry: entry}, nil
}

func (r *Repository) ConfirmReservation(ctx context.Context, requestID string) error {
	refUUID, err := stringToUUID(requestID)
	if err != nil {
		return fmt.Errorf("invalid request id: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	q := repository.New(tx)

	entry, err := q.GetLedgerEntryByReferenceID(ctx, pgtypeUUIDFromUUID(refUUID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrReservationNotFound
		}

		return fmt.Errorf("get reservation: %w", err)
	}

	if entry.Status != repository.LedgerStatusPending {
		return ErrReservationNotPending
	}

	if entry.EntryType != repository.EntryTypeReservation {
		return errors.New("entry reference does not point to a reservation")
	}

	if _, err := q.UpdateLedgerEntryStatus(ctx, repository.UpdateLedgerEntryStatusParams{
		ID:     entry.ID,
		Status: repository.LedgerStatusCompleted,
	}); err != nil {
		return fmt.Errorf("update reservation status: %w", err)
	}

	if _, err := q.InsertLedgerEntry(ctx, repository.InsertLedgerEntryParams{
		AccountID:     entry.AccountID,
		EntryType:     repository.EntryTypeDeduction,
		Amount:        entry.Amount,
		BalanceAfter:  entry.BalanceAfter,
		Currency:      entry.Currency,
		ReferenceID:   entry.ReferenceID,
		ReferenceType: entry.ReferenceType,
		Status:        repository.LedgerStatusCompleted,
		Description:   pgtype.Text{String: "deduction confirmed from reservation", Valid: true},
	}); err != nil {
		return fmt.Errorf("insert deduction entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *Repository) ReleaseReservation(ctx context.Context, requestID string) error {
	refUUID, err := stringToUUID(requestID)
	if err != nil {
		return fmt.Errorf("invalid request id: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	q := repository.New(tx)

	entry, err := q.GetLedgerEntryByReferenceID(ctx, pgtypeUUIDFromUUID(refUUID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrReservationNotFound
		}

		return fmt.Errorf("get reservation: %w", err)
	}

	if entry.Status != repository.LedgerStatusPending {
		return ErrReservationNotPending
	}

	if entry.EntryType != repository.EntryTypeReservation {
		return errors.New("entry reference does not point to a reservation")
	}

	account, err := q.GetAccountByIDForUpdate(ctx, entry.AccountID)
	if err != nil {
		return fmt.Errorf("lock account: %w", err)
	}

	amountDec, err := numericToDecimal(entry.Amount)
	if err != nil {
		return fmt.Errorf("parse entry amount: %w", err)
	}

	balanceDec, err := numericToDecimal(account.Balance)
	if err != nil {
		return fmt.Errorf("parse balance: %w", err)
	}

	refundedBalance := balanceDec.Add(amountDec)
	refundedNumeric, err := decimalToNumeric(refundedBalance)
	if err != nil {
		return fmt.Errorf("convert refunded balance: %w", err)
	}

	if _, err := q.UpdateAccountBalance(ctx, repository.UpdateAccountBalanceParams{
		ID:      account.ID,
		Balance: refundedNumeric,
	}); err != nil {
		return fmt.Errorf("refund balance: %w", err)
	}

	if _, err := q.UpdateLedgerEntryStatus(ctx, repository.UpdateLedgerEntryStatusParams{
		ID:     entry.ID,
		Status: repository.LedgerStatusCancelled,
	}); err != nil {
		return fmt.Errorf("cancel reservation: %w", err)
	}

	if _, err := q.InsertLedgerEntry(ctx, repository.InsertLedgerEntryParams{
		AccountID:     account.ID,
		EntryType:     repository.EntryTypeRefund,
		Amount:        entry.Amount,
		BalanceAfter:  refundedNumeric,
		Currency:      account.Currency,
		ReferenceID:   entry.ReferenceID,
		ReferenceType: entry.ReferenceType,
		Status:        repository.LedgerStatusCompleted,
		Description:   pgtype.Text{String: "refund from cancelled reservation", Valid: true},
	}); err != nil {
		return fmt.Errorf("insert refund entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *Repository) GetOrCreateAccount(ctx context.Context, ownerID uuid.UUID) (repository.Account, error) {
	q := repository.New(r.pool)
	account, err := q.GetOrCreateAccount(ctx, ownerID)
	if err != nil {
		return repository.Account{}, fmt.Errorf("get or create account: %w", err)
	}

	return account, nil
}

func (r *Repository) ListEntriesByAccount(
	ctx context.Context,
	accountID uuid.UUID,
	limit, offset int32,
) ([]repository.LedgerEntry, error) {
	q := repository.New(r.pool)
	entries, err := q.ListLedgerEntriesByAccount(ctx, repository.ListLedgerEntriesByAccountParams{
		AccountID: accountID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list ledger entries: %w", err)
	}

	return entries, nil
}
