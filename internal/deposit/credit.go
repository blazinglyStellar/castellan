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
	Asset       string
}

type CreditHandler struct {
	pool    *pgxpool.Pool
	queries repository.Querier
	cfg     stellar.Config
	log     *slog.Logger
}

func NewCreditHandler(pool *pgxpool.Pool, cfg stellar.Config) *CreditHandler {
	return &CreditHandler{
		pool: pool,
		cfg:  cfg,
		log:  slog.With(slog.String("component", "credit_handler")),
	}
}

func (h *CreditHandler) queriesFn() repository.Querier {
	if h.queries != nil {
		return h.queries
	}
	return repository.New(h.pool)
}

func (h *CreditHandler) CreditDeposit(ctx context.Context, op PaymentOperation) error {
	if _, err := uuid.Parse(op.Memo); err != nil {
		h.log.WarnContext(ctx, "invalid deposit memo",
			slog.String("tx_hash", op.TxHash),
			slog.String("memo_prefix", memoPrefix(op.Memo)),
		)
		return nil
	}

	q := h.queriesFn()

	user, err := q.GetUserByDepositMemo(ctx, pgtype.Text{String: op.Memo, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.log.WarnContext(ctx, "unknown deposit memo",
				slog.String("tx_hash", op.TxHash),
				slog.String("memo_prefix", memoPrefix(op.Memo)),
			)
			return nil
		}
		return fmt.Errorf("lookup deposit memo: %w", err)
	}

	account, err := q.GetOrCreateAccount(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get or create account: %w", err)
	}

	amountNumeric, err := decimalToNumeric(op.Amount)
	if err != nil {
		return fmt.Errorf("convert amount: %w", err)
	}

	if op.Asset != "XLM" {
		h.log.WarnContext(ctx, "non-XLM deposit rejected",
			slog.String("tx_hash", op.TxHash),
			slog.String("asset", op.Asset),
		)
		if _, err := q.InsertDeposit(ctx, repository.InsertDepositParams{
			AccountID:   account.ID,
			FromAddress: op.FromAddress,
			Amount:      amountNumeric,
			Currency:    currencyFromAsset(op.Asset),
			Memo:        pgtype.Text{String: op.Memo, Valid: true},
			TxHash:      op.TxHash,
			Status:      repository.DepositStatusFailed,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				h.log.InfoContext(ctx, "duplicate deposit tx_hash",
					slog.String("tx_hash", op.TxHash),
				)
				return nil
			}
			return fmt.Errorf("insert failed deposit: %w", err)
		}
		return nil
	}

	if op.Amount.LessThan(h.cfg.MinDepositAmount) {
		h.log.WarnContext(ctx, "deposit below minimum",
			slog.String("tx_hash", op.TxHash),
			slog.String("amount", op.Amount.String()),
			slog.String("min_amount", h.cfg.MinDepositAmount.String()),
		)
		if _, err := q.InsertDeposit(ctx, repository.InsertDepositParams{
			AccountID:   account.ID,
			FromAddress: op.FromAddress,
			Amount:      amountNumeric,
			Currency:    repository.CurrencyXLM,
			Memo:        pgtype.Text{String: op.Memo, Valid: true},
			TxHash:      op.TxHash,
			Status:      repository.DepositStatusFailed,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				h.log.InfoContext(ctx, "duplicate deposit tx_hash",
					slog.String("tx_hash", op.TxHash),
				)
				return nil
			}
			return fmt.Errorf("insert failed deposit: %w", err)
		}
		return nil
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tq := repository.New(tx)

	account, err = tq.GetAccountForUpdate(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("lock account: %w", err)
	}

	depositRecord, err := tq.InsertDeposit(ctx, repository.InsertDepositParams{
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
			h.log.InfoContext(ctx, "duplicate deposit tx_hash",
				slog.String("tx_hash", op.TxHash),
			)
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("commit duplicate: %w", err)
			}
			return nil
		}
		return fmt.Errorf("insert deposit: %w", err)
	}

	balanceDec, err := numericToDecimal(account.Balance)
	if err != nil {
		return fmt.Errorf("parse balance: %w", err)
	}

	newBalance := balanceDec.Add(op.Amount)
	newBalanceNumeric, err := decimalToNumeric(newBalance)
	if err != nil {
		return fmt.Errorf("convert new balance: %w", err)
	}

	if _, err := tq.UpdateAccountBalance(ctx, repository.UpdateAccountBalanceParams{
		ID:      account.ID,
		Balance: newBalanceNumeric,
	}); err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	if _, err := tq.InsertLedgerEntry(ctx, repository.InsertLedgerEntryParams{
		AccountID:     account.ID,
		EntryType:     repository.EntryTypeDeposit,
		Amount:        amountNumeric,
		BalanceAfter:  newBalanceNumeric,
		Currency:      repository.CurrencyXLM,
		ReferenceID:   pgtypeUUIDFromUUID(depositRecord.ID),
		ReferenceType: pgtype.Text{String: "deposit", Valid: true},
		Status:        repository.LedgerStatusCompleted,
		Description:   pgtype.Text{String: "deposit credit", Valid: true},
	}); err != nil {
		return fmt.Errorf("insert ledger entry: %w", err)
	}

	if _, err := tq.ConfirmDeposit(ctx, depositRecord.ID); err != nil {
		return fmt.Errorf("confirm deposit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	h.log.LogAttrs(ctx, slog.LevelInfo, "deposit credited",
		slog.String("consumer_id", user.ID.String()),
		slog.String("amount", op.Amount.String()),
		slog.String("outcome", "confirmed"),
		slog.String("tx_hash", op.TxHash),
	)

	return nil
}

func currencyFromAsset(asset string) repository.Currency {
	switch asset {
	case "USDC":
		return repository.CurrencyUSDC
	default:
		return repository.CurrencyXLM
	}
}

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, errors.New("numeric is null")
	}
	if n.NaN {
		return decimal.Zero, errors.New("numeric is NaN")
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

const memoPrefixLen = 8

func memoPrefix(s string) string {
	if len(s) > memoPrefixLen {
		return s[:memoPrefixLen]
	}
	return s
}
