//go:build integration

package deposit

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool
var testQueries *repository.Queries

func TestMain(m *testing.M) {
	teardown, err := mustStartPostgresContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not start postgres container: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if testPool != nil {
		testPool.Close()
	}

	if teardown != nil {
		if err := teardown(context.Background()); err != nil {
			slog.Warn("could not teardown postgres container", slog.String("error", err.Error()))
		}
	}

	os.Exit(code)
}

func mustStartPostgresContainer() (func(context.Context, ...testcontainers.TerminateOption) error, error) {
	ctx := context.Background()

	dbContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("castellan_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			tcwait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not start postgres container: %w", err)
	}

	host, err := dbContainer.Host(ctx)
	if err != nil {
		return dbContainer.Terminate, fmt.Errorf("could not get host: %w", err)
	}

	port, err := dbContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return dbContainer.Terminate, fmt.Errorf("could not get port: %w", err)
	}

	connStr := fmt.Sprintf("postgres://test:test@%s:%s/castellan_test?sslmode=disable", host, port.Port())

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return dbContainer.Terminate, fmt.Errorf("could not create pool: %w", err)
	}

	testPool = pool
	testQueries = repository.New(pool)

	if _, err := pool.Exec(ctx, depositMigrationSQL); err != nil {
		return dbContainer.Terminate, fmt.Errorf("could not run migration: %w", err)
	}

	return dbContainer.Terminate, nil
}

const depositMigrationSQL = `
CREATE TYPE currency AS ENUM ('XLM', 'USDC');
CREATE TYPE entry_type AS ENUM ('deposit', 'reservation', 'deduction', 'refund', 'settlement');
CREATE TYPE ledger_status AS ENUM ('pending', 'completed', 'failed', 'cancelled');
CREATE TYPE deposit_status AS ENUM ('pending', 'confirmed', 'failed');

CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                   TEXT NOT NULL UNIQUE,
    deposit_memo            TEXT UNIQUE,
    payout_stellar_address  TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_deposit_memo ON users (deposit_memo);

CREATE TABLE accounts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance     NUMERIC(20,10) NOT NULL DEFAULT 0,
    currency    currency NOT NULL DEFAULT 'XLM',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ledger_entries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    entry_type    entry_type NOT NULL,
    amount        NUMERIC(20,10) NOT NULL,
    balance_after NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    reference_id  UUID,
    reference_type TEXT,
    status        ledger_status NOT NULL DEFAULT 'completed',
    description   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE deposits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    from_address    TEXT NOT NULL,
    amount          NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    memo            TEXT,
    tx_hash         TEXT NOT NULL UNIQUE,
    status          deposit_status NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at    TIMESTAMPTZ
);

CREATE INDEX idx_deposits_account ON deposits (account_id);
CREATE INDEX idx_deposits_tx_hash ON deposits (tx_hash);

ALTER TABLE deposits ADD COLUMN IF NOT EXISTS reason TEXT;
`

func seedCreditTestUser(ctx context.Context, t *testing.T, balance string) (uuid.UUID, uuid.UUID, string) {
	t.Helper()

	userID := uuid.New()
	memo := uuid.New().String()
	email := fmt.Sprintf("credit-%s@example.com", userID.String())

	_, err := testPool.Exec(ctx,
		`INSERT INTO users (id, email, deposit_memo) VALUES ($1, $2, $3)`,
		userID, email, memo)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	accountID := uuid.New()
	_, err = testPool.Exec(ctx,
		`INSERT INTO accounts (id, owner_id, balance) VALUES ($1, $2, $3)`,
		accountID, userID, balance)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}

	return userID, accountID, memo
}

func creditTestCfg() stellar.Config {
	return stellar.Config{
		MinDepositAmount: decimal.NewFromInt(5),
	}
}

func TestCreditDepositIntegration_ValidPayment(t *testing.T) {
	ctx := context.Background()
	userID, accountID, memo := seedCreditTestUser(ctx, t, "100.00")

	handler := NewCreditHandler(testPool, creditTestCfg())
	op := PaymentOperation{
		TxHash:      "tx-valid-" + uuid.New().String(),
		FromAddress: "GBBBBB",
		Amount:      decimal.NewFromInt(10),
		Memo:        memo,
		Asset:       "XLM",
	}

	err := handler.CreditDeposit(ctx, op)
	if err != nil {
		t.Fatalf("CreditDeposit: %v", err)
	}

	// Verify deposit was confirmed
	deposits, err := testQueries.ListDepositsByAccount(ctx, accountID)
	if err != nil {
		t.Fatalf("list deposits: %v", err)
	}
	if len(deposits) != 1 {
		t.Fatalf("expected 1 deposit, got %d", len(deposits))
	}
	if deposits[0].Status != repository.DepositStatusConfirmed {
		t.Errorf("status = %q, want %q", deposits[0].Status, repository.DepositStatusConfirmed)
	}
	if !deposits[0].ConfirmedAt.Valid {
		t.Error("confirmed_at is null, should be set")
	}

	// Verify balance increased
	account, err := testQueries.GetAccountBalance(ctx, userID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	actualBalance := numericToBigFloat(account)
	if actualBalance != 110.00 {
		t.Errorf("balance = %.2f, want %.2f", actualBalance, 110.00)
	}

	// Verify ledger entry was created
	var entryCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM ledger_entries WHERE account_id = $1 AND entry_type = 'deposit'`, accountID).Scan(&entryCount)
	if err != nil {
		t.Fatalf("count ledger entries: %v", err)
	}
	if entryCount != 1 {
		t.Errorf("expected 1 ledger entry, got %d", entryCount)
	}
}

func TestCreditDepositIntegration_DuplicateTxHash(t *testing.T) {
	ctx := context.Background()
	userID, accountID, memo := seedCreditTestUser(ctx, t, "100.00")

	handler := NewCreditHandler(testPool, creditTestCfg())
	txHash := "tx-dup-" + uuid.New().String()
	op := PaymentOperation{
		TxHash:      txHash,
		FromAddress: "GBBBBB",
		Amount:      decimal.NewFromInt(10),
		Memo:        memo,
		Asset:       "XLM",
	}

	if err := handler.CreditDeposit(ctx, op); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// Second call with same tx_hash — should be silently ignored
	if err := handler.CreditDeposit(ctx, op); err != nil {
		t.Fatalf("second call: %v", err)
	}

	// Only one deposit row should exist
	deposits, err := testQueries.ListDepositsByAccount(ctx, accountID)
	if err != nil {
		t.Fatalf("list deposits: %v", err)
	}
	if len(deposits) != 1 {
		t.Fatalf("expected 1 deposit, got %d", len(deposits))
	}

	// Balance should only be credited once (100 + 10 = 110, not 120)
	balance, err := testQueries.GetAccountBalance(ctx, userID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	bd := numericToBigDecimal(balance)
	if !bd.Equal(decimal.NewFromInt(110)) {
		t.Errorf("balance = %s, want 110", bd.String())
	}
}

func TestCreditDepositIntegration_UnknownMemo(t *testing.T) {
	ctx := context.Background()
	seedCreditTestUser(ctx, t, "100.00")

	handler := NewCreditHandler(testPool, creditTestCfg())
	op := PaymentOperation{
		TxHash:      "tx-unknown-" + uuid.New().String(),
		FromAddress: "GBBBBB",
		Amount:      decimal.NewFromInt(10),
		Memo:        uuid.New().String(), // not seeded for any user
		Asset:       "XLM",
	}

	err := handler.CreditDeposit(ctx, op)
	if err != nil {
		t.Fatalf("CreditDeposit: %v", err)
	}

	// Deposit should be inserted as failed (via InsertDeposit called from unknown memo path)
	deposit, err := testQueries.GetDepositByTxHash(ctx, op.TxHash)
	if err != nil {
		if err == pgx.ErrNoRows {
			t.Fatal("expected a deposit row to be inserted for unknown memo")
		}
		t.Fatalf("get deposit: %v", err)
	}
	if deposit.Status != repository.DepositStatusFailed {
		t.Errorf("status = %q, want %q", deposit.Status, repository.DepositStatusFailed)
	}
}

func TestCreditDepositIntegration_NonXLM(t *testing.T) {
	ctx := context.Background()
	_, accountID, memo := seedCreditTestUser(ctx, t, "100.00")

	handler := NewCreditHandler(testPool, creditTestCfg())
	op := PaymentOperation{
		TxHash:      "tx-nonxlm-" + uuid.New().String(),
		FromAddress: "GBBBBB",
		Amount:      decimal.NewFromInt(10),
		Memo:        memo,
		Asset:       "USDC",
	}

	err := handler.CreditDeposit(ctx, op)
	if err != nil {
		t.Fatalf("CreditDeposit: %v", err)
	}

	deposits, err := testQueries.ListDepositsByAccount(ctx, accountID)
	if err != nil {
		t.Fatalf("list deposits: %v", err)
	}
	if len(deposits) != 1 {
		t.Fatalf("expected 1 deposit, got %d", len(deposits))
	}
	if deposits[0].Status != repository.DepositStatusFailed {
		t.Errorf("status = %q, want %q", deposits[0].Status, repository.DepositStatusFailed)
	}
	if deposits[0].Currency != repository.CurrencyUSDC {
		t.Errorf("currency = %q, want %q", deposits[0].Currency, repository.CurrencyUSDC)
	}
}

func TestCreditDepositIntegration_BelowMinimum(t *testing.T) {
	ctx := context.Background()
	_, accountID, memo := seedCreditTestUser(ctx, t, "100.00")

	handler := NewCreditHandler(testPool, creditTestCfg())
	op := PaymentOperation{
		TxHash:      "tx-belowmin-" + uuid.New().String(),
		FromAddress: "GBBBBB",
		Amount:      decimal.NewFromInt(1), // below MinDepositAmount of 5
		Memo:        memo,
		Asset:       "XLM",
	}

	err := handler.CreditDeposit(ctx, op)
	if err != nil {
		t.Fatalf("CreditDeposit: %v", err)
	}

	deposits, err := testQueries.ListDepositsByAccount(ctx, accountID)
	if err != nil {
		t.Fatalf("list deposits: %v", err)
	}
	if len(deposits) != 1 {
		t.Fatalf("expected 1 deposit, got %d", len(deposits))
	}
	if deposits[0].Status != repository.DepositStatusFailed {
		t.Errorf("status = %q, want %q", deposits[0].Status, repository.DepositStatusFailed)
	}

	// Balance should remain unchanged
	balanceRow, err := testPool.Query(ctx, `SELECT balance FROM accounts WHERE id = $1`, accountID)
	if err != nil {
		t.Fatalf("query balance: %v", err)
	}
	defer balanceRow.Close()
	if balanceRow.Next() {
		var balance pgtype.Numeric
		if err := balanceRow.Scan(&balance); err != nil {
			t.Fatalf("scan balance: %v", err)
		}
		bd := numericToBigDecimal(balance)
		if !bd.Equal(decimal.NewFromInt(100)) {
			t.Errorf("balance = %s, want 100", bd.String())
		}
	}
}

func numericToBigFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	d := decimal.NewFromBigInt(n.Int, n.Exp)
	f, _ := d.Float64()
	return f
}

func numericToBigDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(n.Int, n.Exp)
}


