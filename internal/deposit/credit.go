package deposit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"
)

type PaymentOperation struct {
	TxHash      string
	FromAddress string
	Amount      decimal.Decimal
	Memo        string
	AssetType   string
}

type creditAccountParams struct {
	amount        decimal.Decimal
	amountNumeric pgtype.Numeric
	depositID     uuid.UUID
	accountID     uuid.UUID
	ownerID       uuid.UUID
}

type CreditHandler struct {
	pool    pgxBeginner
	queries repository.Querier
	cfg     stellar.Config
	logger  *slog.Logger
}

type pgxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func NewCreditHandler(pool *pgxpool.Pool, queries repository.Querier, cfg stellar.Config, logger *slog.Logger) *CreditHandler {
	return &CreditHandler{pool: pool, queries: queries, cfg: cfg, logger: logger}
}

func (h *CreditHandler) CreditDeposit(ctx context.Context, op PaymentOperation) error {
	logger := h.logger.With(
		slog.String("tx_hash", op.TxHash),
		slog.String("amount", op.Amount.String()),
	)

	memoUUID, amount, err := h.validate(op)
	if err != nil {
		logger.WarnContext(ctx, "deposit validation failed",
			slog.String("reason", err.Error()),
			slog.String("outcome", "skipped"),
		)
		return nil
	}

	user, err := h.queries.GetUserByDepositMemo(ctx, pgtype.Text{String: memoUUID.String(), Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.WarnContext(ctx, "deposit memo unknown",
				slog.String("memo", op.Memo),
				slog.String("outcome", "skipped"),
			)
			return nil
		}
		return fmt.Errorf("get user by memo: %w", err)
	}

	account, err := h.queries.GetOrCreateAccount(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get or create account: %w", err)
	}

	amountNumeric, err := decimalToNumeric(amount)
	if err != nil {
		return fmt.Errorf("convert amount: %w", err)
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := repository.New(tx)

	deposit, err := q.InsertDepositIgnoreConflict(ctx, repository.InsertDepositIgnoreConflictParams{
		AccountID:   account.ID,
		FromAddress: op.FromAddress,
		Amount:      amountNumeric,
		Currency:    repository.CurrencyXLM,
		Memo:        pgtype.Text{String: op.Memo, Valid: true},
		TxHash:      op.TxHash,
		Status:      repository.DepositStatusPending,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.InfoContext(ctx, "duplicate deposit skipped",
				slog.String("outcome", "duplicate"),
			)
			return nil
		}
		return fmt.Errorf("insert deposit: %w", err)
	}

	p := creditAccountParams{
		amount:        amount,
		amountNumeric: amountNumeric,
		depositID:     deposit.ID,
		accountID:     account.ID,
		ownerID:       user.ID,
	}

	if err := h.creditAccount(ctx, q, p); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	logger.InfoContext(ctx, "deposit credited",
		slog.String("consumer_id", user.ID.String()),
		slog.String("outcome", "confirmed"),
	)

	return nil
}

func (h *CreditHandler) creditAccount(ctx context.Context, q repository.Querier, p creditAccountParams) error {
	locked, err := q.GetAccountForUpdate(ctx, p.ownerID)
	if err != nil {
		return fmt.Errorf("lock account: %w", err)
	}

	balanceDec, err := numericToDecimal(locked.Balance)
	if err != nil {
		return fmt.Errorf("parse balance: %w", err)
	}

	newBalance := balanceDec.Add(p.amount)
	newBalanceNumeric, err := decimalToNumeric(newBalance)
	if err != nil {
		return fmt.Errorf("convert new balance: %w", err)
	}

	if _, err := q.UpdateAccountBalance(ctx, repository.UpdateAccountBalanceParams{
		ID:      locked.ID,
		Balance: newBalanceNumeric,
	}); err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	if _, err := q.InsertLedgerEntry(ctx, repository.InsertLedgerEntryParams{
		AccountID:     p.accountID,
		EntryType:     repository.EntryTypeDeposit,
		Amount:        p.amountNumeric,
		BalanceAfter:  newBalanceNumeric,
		Currency:      repository.CurrencyXLM,
		ReferenceID:   pgtypeUUIDFromUUID(p.depositID),
		ReferenceType: pgtype.Text{String: "deposit", Valid: true},
		Status:        repository.LedgerStatusCompleted,
		Description:   pgtype.Text{String: "deposit credited from Stellar payment", Valid: true},
	}); err != nil {
		return fmt.Errorf("insert ledger entry: %w", err)
	}

	if _, err := q.ConfirmDeposit(ctx, p.depositID); err != nil {
		return fmt.Errorf("confirm deposit: %w", err)
	}

	return nil
}

func (h *CreditHandler) validate(op PaymentOperation) (uuid.UUID, decimal.Decimal, error) {
	if op.AssetType != "native" {
		return uuid.Nil, decimal.Zero, errors.New("non-native asset")
	}

	if op.Amount.LessThan(decimal.Zero) {
		return uuid.Nil, decimal.Zero, errors.New("negative amount")
	}

	memoUUID, err := uuid.Parse(op.Memo)
	if err != nil {
		return uuid.Nil, decimal.Zero, fmt.Errorf("invalid memo: %w", err)
	}

	if op.Amount.LessThan(h.cfg.MinDepositAmount) {
		return uuid.Nil, decimal.Zero, fmt.Errorf("below minimum %s", h.cfg.MinDepositAmount.String())
	}

	return memoUUID, op.Amount, nil
}

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, errors.New("numeric is null")
	}
	if n.NaN {
		return decimal.Zero, errors.New("numeric is NaN")
	}
	if n.InfinityModifier != 0 {
		return decimal.Zero, errors.New("numeric is infinite")
	}
	return decimal.NewFromBigInt(n.Int, n.Exp), nil
}

func decimalToNumeric(d decimal.Decimal) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(d.String()); err != nil {
		return n, fmt.Errorf("scan decimal to numeric: %w", err)
	}
	return n, nil
}

func pgtypeUUIDFromUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}
