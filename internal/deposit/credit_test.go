package deposit

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"
)

func TestCreditDeposit_NonNativeAsset(t *testing.T) {
	t.Parallel()

	h := &CreditHandler{logger: slog.Default()}
	err := h.CreditDeposit(context.Background(), PaymentOperation{
		TxHash:    "abc",
		Amount:    decimal.NewFromInt(10),
		Memo:      uuid.New().String(),
		AssetType: "not-native",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCreditDeposit_InvalidMemoFormat(t *testing.T) {
	t.Parallel()

	h := &CreditHandler{logger: slog.Default()}
	err := h.CreditDeposit(context.Background(), PaymentOperation{
		TxHash:    "abc",
		Amount:    decimal.NewFromInt(10),
		Memo:      "not-a-uuid",
		AssetType: "native",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCreditDeposit_UnknownMemo(t *testing.T) {
	t.Parallel()

	mq := &creditMockQuerier{
		getUserByDepositMemo: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{}, pgx.ErrNoRows
		},
	}
	h := &CreditHandler{queries: mq, cfg: defaultCfg(), logger: slog.Default()}

	err := h.CreditDeposit(context.Background(), PaymentOperation{
		TxHash:    "abc",
		Amount:    decimal.NewFromInt(10),
		Memo:      uuid.New().String(),
		AssetType: "native",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCreditDeposit_BelowMinimum(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	mq := &creditMockQuerier{
		getUserByDepositMemo: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{ID: userID}, nil
		},
	}
	h := &CreditHandler{
		queries: mq,
		cfg:     stellar.Config{MinDepositAmount: decimal.NewFromInt(5)},
		logger:  slog.Default(),
	}

	err := h.CreditDeposit(context.Background(), PaymentOperation{
		TxHash:    "abc",
		Amount:    decimal.NewFromInt(1),
		Memo:      uuid.New().String(),
		AssetType: "native",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCreditDeposit_GetUserByMemoDBError(t *testing.T) {
	t.Parallel()

	mq := &creditMockQuerier{
		getUserByDepositMemo: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{}, errors.New("connection refused")
		},
	}
	h := &CreditHandler{queries: mq, cfg: defaultCfg(), logger: slog.Default()}

	err := h.CreditDeposit(context.Background(), PaymentOperation{
		TxHash:    "abc",
		Amount:    decimal.NewFromInt(10),
		Memo:      uuid.New().String(),
		AssetType: "native",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreditDeposit_BeginTxError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	accountID := uuid.New()
	mq := &creditMockQuerier{
		getUserByDepositMemo: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{ID: userID}, nil
		},
		getOrCreateAccount: func(_ context.Context, ownerID uuid.UUID) (repository.Account, error) {
			return repository.Account{ID: accountID, OwnerID: ownerID}, nil
		},
	}
	fb := &fakeBeginner{beginErr: errors.New("pool closed")}
	h := &CreditHandler{
		pool:    fb,
		queries: mq,
		cfg:     defaultCfg(),
		logger:  slog.Default(),
	}

	err := h.CreditDeposit(context.Background(), PaymentOperation{
		TxHash:    "abc",
		Amount:    decimal.NewFromInt(10),
		Memo:      uuid.New().String(),
		AssetType: "native",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreditDeposit_DuplicateTxHash(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	accountID := uuid.New()
	mq := &creditMockQuerier{
		getUserByDepositMemo: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{ID: userID}, nil
		},
		getOrCreateAccount: func(_ context.Context, ownerID uuid.UUID) (repository.Account, error) {
			return repository.Account{ID: accountID, OwnerID: ownerID}, nil
		},
	}
	fb := &fakeBeginner{
		tx: &fakeTx{
			queryRowErr: pgx.ErrNoRows,
		},
	}
	h := &CreditHandler{
		pool:    fb,
		queries: mq,
		cfg:     defaultCfg(),
		logger:  slog.Default(),
	}

	err := h.CreditDeposit(context.Background(), PaymentOperation{
		TxHash:    "abc",
		Amount:    decimal.NewFromInt(10),
		Memo:      uuid.New().String(),
		AssetType: "native",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCreditDeposit_InsertDepositError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	accountID := uuid.New()
	mq := &creditMockQuerier{
		getUserByDepositMemo: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{ID: userID}, nil
		},
		getOrCreateAccount: func(_ context.Context, ownerID uuid.UUID) (repository.Account, error) {
			return repository.Account{ID: accountID, OwnerID: ownerID}, nil
		},
	}
	fb := &fakeBeginner{
		tx: &fakeTx{queryRowErr: errors.New("constraint violation")},
	}
	h := &CreditHandler{
		pool:    fb,
		queries: mq,
		cfg:     defaultCfg(),
		logger:  slog.Default(),
	}

	err := h.CreditDeposit(context.Background(), PaymentOperation{
		TxHash:    "abc",
		Amount:    decimal.NewFromInt(10),
		Memo:      uuid.New().String(),
		AssetType: "native",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreditAccount_HappyPath(t *testing.T) {
	t.Parallel()

	depositID := uuid.New()
	accountID := uuid.New()
	ownerID := uuid.New()
	amount := decimal.NewFromInt(10)

	mq := &creditMockQuerier{
		getAccountForUpdate: func(_ context.Context, _ uuid.UUID) (repository.Account, error) {
			return repository.Account{
				ID:      accountID,
				OwnerID: ownerID,
				Balance: decimalToNumericSafe(decimal.Zero),
			}, nil
		},
		updateAccountBalance: func(_ context.Context, p repository.UpdateAccountBalanceParams) (repository.Account, error) {
			return repository.Account{ID: p.ID, Balance: p.Balance}, nil
		},
		insertLedgerEntry: func(_ context.Context, p repository.InsertLedgerEntryParams) (repository.LedgerEntry, error) {
			return repository.LedgerEntry{
				AccountID:    p.AccountID,
				EntryType:    p.EntryType,
				Amount:       p.Amount,
				BalanceAfter: p.BalanceAfter,
				ReferenceID:  p.ReferenceID,
			}, nil
		},
		confirmDeposit: func(_ context.Context, id uuid.UUID) (repository.Deposit, error) {
			return repository.Deposit{ID: id, Status: repository.DepositStatusConfirmed}, nil
		},
	}

	h := &CreditHandler{logger: slog.Default()}
	err := h.creditAccount(context.Background(), mq, creditAccountParams{
		amount:        amount,
		amountNumeric: decimalToNumericSafe(amount),
		depositID:     depositID,
		accountID:     accountID,
		ownerID:       ownerID,
	})
	if err != nil {
		t.Fatalf("creditAccount: %v", err)
	}
}

func TestCreditAccount_GetAccountForUpdateError(t *testing.T) {
	t.Parallel()

	mq := &creditMockQuerier{
		getAccountForUpdate: func(_ context.Context, _ uuid.UUID) (repository.Account, error) {
			return repository.Account{}, errors.New("lock timeout")
		},
	}

	h := &CreditHandler{logger: slog.Default()}
	err := h.creditAccount(context.Background(), mq, creditAccountParams{
		ownerID: uuid.New(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- mocks ---

type creditMockQuerier struct {
	repository.Querier
	getUserByDepositMemo func(context.Context, pgtype.Text) (repository.User, error)
	getOrCreateAccount   func(context.Context, uuid.UUID) (repository.Account, error)
	getAccountForUpdate  func(context.Context, uuid.UUID) (repository.Account, error)
	updateAccountBalance func(context.Context, repository.UpdateAccountBalanceParams) (repository.Account, error)
	insertLedgerEntry    func(context.Context, repository.InsertLedgerEntryParams) (repository.LedgerEntry, error)
	confirmDeposit       func(context.Context, uuid.UUID) (repository.Deposit, error)
}

func (m *creditMockQuerier) GetUserByDepositMemo(ctx context.Context, memo pgtype.Text) (repository.User, error) {
	if m.getUserByDepositMemo == nil {
		return repository.User{}, errors.New("unexpected call to GetUserByDepositMemo")
	}
	return m.getUserByDepositMemo(ctx, memo)
}

func (m *creditMockQuerier) GetOrCreateAccount(ctx context.Context, ownerID uuid.UUID) (repository.Account, error) {
	if m.getOrCreateAccount == nil {
		return repository.Account{}, errors.New("unexpected call to GetOrCreateAccount")
	}
	return m.getOrCreateAccount(ctx, ownerID)
}

func (m *creditMockQuerier) GetAccountForUpdate(ctx context.Context, ownerID uuid.UUID) (repository.Account, error) {
	if m.getAccountForUpdate == nil {
		return repository.Account{}, errors.New("unexpected call to GetAccountForUpdate")
	}
	return m.getAccountForUpdate(ctx, ownerID)
}

func (m *creditMockQuerier) UpdateAccountBalance(ctx context.Context, arg repository.UpdateAccountBalanceParams) (repository.Account, error) {
	if m.updateAccountBalance == nil {
		return repository.Account{}, errors.New("unexpected call to UpdateAccountBalance")
	}
	return m.updateAccountBalance(ctx, arg)
}

func (m *creditMockQuerier) InsertLedgerEntry(ctx context.Context, arg repository.InsertLedgerEntryParams) (repository.LedgerEntry, error) {
	if m.insertLedgerEntry == nil {
		return repository.LedgerEntry{}, errors.New("unexpected call to InsertLedgerEntry")
	}
	return m.insertLedgerEntry(ctx, arg)
}

func (m *creditMockQuerier) ConfirmDeposit(ctx context.Context, id uuid.UUID) (repository.Deposit, error) {
	if m.confirmDeposit == nil {
		return repository.Deposit{}, errors.New("unexpected call to ConfirmDeposit")
	}
	return m.confirmDeposit(ctx, id)
}

type fakeBeginner struct {
	beginErr error
	tx       *fakeTx
}

func (f *fakeBeginner) Begin(_ context.Context) (pgx.Tx, error) {
	if f.beginErr != nil {
		return nil, f.beginErr
	}
	return f.tx, nil
}

type fakeTx struct {
	pgx.Tx
	queryRowErr error
	resultRow   map[string]any
	committed   bool
	rolledBack  bool
}

func (f *fakeTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return &fakeRow{
		err:  f.queryRowErr,
		data: f.resultRow,
	}
}

func (f *fakeTx) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, f.queryRowErr
}

func (f *fakeTx) Commit(_ context.Context) error {
	f.committed = true
	return nil
}

func (f *fakeTx) Rollback(_ context.Context) error {
	f.rolledBack = true
	return nil
}

type fakeRow struct {
	err  error
	data map[string]any
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	// Map field names to their string values from resultRow.
	fields := map[string]string{}
	for k, v := range r.data {
		if s, ok := v.(string); ok {
			fields[k] = s
		}
	}

	for _, d := range dest {
		switch v := d.(type) {
		case *uuid.UUID:
			if id, ok := fields["id"]; ok {
				parsed, err := uuid.Parse(id)
				if err != nil {
					return err
				}
				*v = parsed
			}
		case *string:
			// handled by mapping below if needed, skip for positional scans
		case *pgtype.Numeric:
			for _, key := range []string{"amount", "balance_after"} {
				if val, ok := r.data[key]; ok {
					if err := v.Scan(val); err != nil {
						return err
					}
					break
				}
			}
		case *pgtype.UUID:
			for _, key := range []string{"reference_id", "account_id"} {
				if val, ok := r.data[key]; ok {
					if err := v.Scan(val); err != nil {
						return err
					}
					break
				}
			}
		case *pgtype.Text:
			for _, key := range []string{"memo", "reference_type", "description"} {
				if val, ok := r.data[key]; ok {
					if err := v.Scan(val); err != nil {
						return err
					}
					break
				}
			}
		case *time.Time:
		case *pgtype.Timestamptz:
		}
	}
	return nil
}

// --- helpers ---

func defaultCfg() stellar.Config {
	return stellar.Config{MinDepositAmount: decimal.NewFromInt(5)}
}

func decimalToNumericSafe(d decimal.Decimal) pgtype.Numeric {
	n, _ := decimalToNumeric(d)
	return n
}
