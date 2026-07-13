package deposit

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/oklog/ulid/v2"
	"github.com/shopspring/decimal"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"
)

var testCfg = stellar.Config{MinDepositAmount: decimal.NewFromInt(5)}

func validOp() PaymentOperation {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	return PaymentOperation{
		TxHash:      "abc123",
		FromAddress: "GBBBBB",
		Amount:      decimal.NewFromInt(10),
		Memo:        ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String(),
		Asset:       "XLM",
	}
}

type mockCreditQuerier struct {
	repository.Querier
	getUserByDepositMemoFunc func(context.Context, pgtype.Text) (repository.User, error)
	insertDepositFunc        func(context.Context, repository.InsertDepositParams) (repository.Deposit, error)
}

func (m *mockCreditQuerier) GetUserByDepositMemo(ctx context.Context, memo pgtype.Text) (repository.User, error) {
	if m.getUserByDepositMemoFunc != nil {
		return m.getUserByDepositMemoFunc(ctx, memo)
	}
	return repository.User{}, errors.New("unexpected call to GetUserByDepositMemo")
}

func (m *mockCreditQuerier) InsertDeposit(ctx context.Context, arg repository.InsertDepositParams) (repository.Deposit, error) {
	if m.insertDepositFunc != nil {
		return m.insertDepositFunc(ctx, arg)
	}
	return repository.Deposit{}, errors.New("unexpected call to InsertDeposit")
}

func testHandler(q repository.Querier) *CreditHandler {
	return &CreditHandler{
		queries: q,
		cfg:     testCfg,
		log:     testLogger(),
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestCreditDeposit_InvalidMemo(t *testing.T) {
	t.Parallel()

	h := testHandler(nil)
	op := validOp()
	op.Memo = "0123456789012345678901234567" // > 26 chars

	err := h.CreditDeposit(context.Background(), op)
	if err != nil {
		t.Fatalf("expected nil for invalid memo, got %v", err)
	}
}

func TestCreditDeposit_UnknownMemo(t *testing.T) {
	t.Parallel()

	q := &mockCreditQuerier{
		getUserByDepositMemoFunc: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{}, pgx.ErrNoRows
		},
	}
	h := testHandler(q)

	err := h.CreditDeposit(context.Background(), validOp())
	if err != nil {
		t.Fatalf("expected nil for unknown memo, got %v", err)
	}
}

func TestCreditDeposit_UserLookupError(t *testing.T) {
	t.Parallel()

	q := &mockCreditQuerier{
		getUserByDepositMemoFunc: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{}, errors.New("db error")
		},
	}
	h := testHandler(q)

	err := h.CreditDeposit(context.Background(), validOp())
	if err == nil {
		t.Fatal("expected error for db failure, got nil")
	}
}

func TestCreditDeposit_NonXlmAsset(t *testing.T) {
	t.Parallel()

	var inserted bool
	q := &mockCreditQuerier{
		getUserByDepositMemoFunc: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{ID: uuid.New()}, nil
		},
		insertDepositFunc: func(_ context.Context, arg repository.InsertDepositParams) (repository.Deposit, error) {
			inserted = true
			if arg.Status != repository.DepositStatusFailed {
				t.Errorf("status = %q, want %q", arg.Status, repository.DepositStatusFailed)
			}
			return repository.Deposit{}, nil
		},
	}
	h := testHandler(q)

	op := validOp()
	op.Asset = "USDC"

	err := h.CreditDeposit(context.Background(), op)
	if err != nil {
		t.Fatalf("expected nil for non-XLM asset, got %v", err)
	}
	if !inserted {
		t.Error("expected failed deposit to be inserted")
	}
}

func TestCreditDeposit_BelowMinAmount(t *testing.T) {
	t.Parallel()

	var inserted bool
	q := &mockCreditQuerier{
		getUserByDepositMemoFunc: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{ID: uuid.New()}, nil
		},
		insertDepositFunc: func(_ context.Context, arg repository.InsertDepositParams) (repository.Deposit, error) {
			inserted = true
			if arg.Status != repository.DepositStatusFailed {
				t.Errorf("status = %q, want %q", arg.Status, repository.DepositStatusFailed)
			}
			return repository.Deposit{}, nil
		},
	}
	h := testHandler(q)

	op := validOp()
	op.Amount = decimal.NewFromInt(1)

	err := h.CreditDeposit(context.Background(), op)
	if err != nil {
		t.Fatalf("expected nil for below-min amount, got %v", err)
	}
	if !inserted {
		t.Error("expected failed deposit to be inserted")
	}
}

func TestCreditDeposit_InsertFailedDepositError(t *testing.T) {
	t.Parallel()

	q := &mockCreditQuerier{
		getUserByDepositMemoFunc: func(_ context.Context, _ pgtype.Text) (repository.User, error) {
			return repository.User{ID: uuid.New()}, nil
		},
		insertDepositFunc: func(_ context.Context, _ repository.InsertDepositParams) (repository.Deposit, error) {
			return repository.Deposit{}, errors.New("insert failed")
		},
	}
	h := testHandler(q)

	op := validOp()
	op.Asset = "USDC"

	err := h.CreditDeposit(context.Background(), op)
	if err == nil {
		t.Fatal("expected error when insert failed deposit fails, got nil")
	}
}

func TestCreditDeposit_LogSanitization(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	q := &mockCreditQuerier{
		getUserByDepositMemoFunc: func(_ context.Context, memo pgtype.Text) (repository.User, error) {
			return repository.User{}, pgx.ErrNoRows
		},
	}
	h := &CreditHandler{
		queries: q,
		cfg:     testCfg,
		log:     logger,
	}

	op := validOp()
	op.Memo = "0123456789012345678901234567"

	_ = h.CreditDeposit(context.Background(), op)

	output := buf.String()
	forbidden := []string{"WALLET_SECRET_KEY", "SecretKey", "secret_key"}
	for _, s := range forbidden {
		if strings.Contains(output, s) {
			t.Errorf("log output contains forbidden string %q", s)
		}
	}
}
