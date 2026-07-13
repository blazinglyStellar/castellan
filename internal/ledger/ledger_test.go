//go:build integration

package ledger_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"castellan/internal/ledger"
	"castellan/internal/repository/db"
)

var (
	testPool    *pgxpool.Pool
	testQueries *repository.Queries
)

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
			fmt.Fprintf(os.Stderr, "could not teardown postgres container: %v\n", err)
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
			wait.ForLog("database system is ready to accept connections").
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
		return dbContainer.Terminate, fmt.Errorf("could not connect to postgres: %w", err)
	}

	testPool = pool
	testQueries = repository.New(pool)

	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		return dbContainer.Terminate, fmt.Errorf("could not run migration: %w", err)
	}

	return dbContainer.Terminate, nil
}

const migrationSQL = `
CREATE TYPE currency AS ENUM ('XLM', 'USDC');
CREATE TYPE entry_type AS ENUM ('deposit', 'reservation', 'deduction', 'refund', 'settlement');
CREATE TYPE ledger_status AS ENUM ('pending', 'completed', 'failed', 'cancelled');

CREATE TABLE users (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                TEXT NOT NULL UNIQUE,
    deposit_memo         TEXT UNIQUE,
    payout_stellar_address TEXT,
    balance              NUMERIC(20,10) NOT NULL DEFAULT 0,
    currency             currency NOT NULL DEFAULT 'XLM',
    account_updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ledger_entries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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

CREATE INDEX idx_ledger_user ON ledger_entries (user_id);
CREATE INDEX idx_ledger_reference ON ledger_entries (reference_id);
`

type testSeed struct {
	UserID uuid.UUID
}

func seedTestData(ctx context.Context, t *testing.T, balance string) testSeed {
	t.Helper()

	userID := uuid.New()
	email := fmt.Sprintf("test-%s@example.com", userID.String())
	_, err := testPool.Exec(ctx, `INSERT INTO users (id, email, balance) VALUES ($1, $2, $3)`, userID, email, balance)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	return testSeed{UserID: userID}
}

func seedUserOnly(ctx context.Context, t *testing.T) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	email := fmt.Sprintf("test-nobal-%s@example.com", userID.String())
	_, err := testPool.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, userID, email)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	return userID
}

func TestReserveBalance_Success(t *testing.T) {
	ctx := context.Background()
	seed := seedTestData(ctx, t, "100.00")

	repo := ledger.NewPostgresRepository(testPool)
	requestID := uuid.NewString()

	result, err := repo.ReserveBalance(ctx, seed.UserID, decimal.NewFromFloat(50), requestID)
	if err != nil {
		t.Fatalf("ReserveBalance: %v", err)
	}

	if result.Entry.UserID != seed.UserID {
		t.Errorf("expected user %s, got %s", seed.UserID, result.Entry.UserID)
	}

	if result.Entry.EntryType != repository.EntryTypeReservation {
		t.Errorf("expected entry type reservation, got %s", result.Entry.EntryType)
	}

	if result.Entry.Status != repository.LedgerStatusPending {
		t.Errorf("expected status pending, got %s", result.Entry.Status)
	}

	entryAmt, _ := result.Entry.Amount.Float64Value()
	if !entryAmt.Valid || entryAmt.Float64 != 50 {
		t.Errorf("expected entry amount 50, got %v", entryAmt.Float64)
	}
}

func TestReserveBalance_InsufficientBalance(t *testing.T) {
	ctx := context.Background()
	seed := seedTestData(ctx, t, "10.00")

	repo := ledger.NewPostgresRepository(testPool)
	requestID := uuid.NewString()

	_, err := repo.ReserveBalance(ctx, seed.UserID, decimal.NewFromFloat(50), requestID)
	if err != ledger.ErrInsufficientBalance {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}

	user, err := testQueries.GetUserByID(ctx, seed.UserID)
	if err != nil {
		t.Fatalf("fetch user: %v", err)
	}

	bal, _ := user.Balance.Float64Value()
	if !bal.Valid || bal.Float64 != 10 {
		t.Errorf("expected balance unchanged at 10, got %v", bal.Float64)
	}
}

func TestReserveBalance_AutoCreateAccount(t *testing.T) {
	ctx := context.Background()
	userID := seedUserOnly(ctx, t)

	repo := ledger.NewPostgresRepository(testPool)
	requestID := uuid.NewString()

	_, err := repo.ReserveBalance(ctx, userID, decimal.NewFromFloat(50), requestID)
	if err != ledger.ErrInsufficientBalance {
		t.Fatalf("expected ErrInsufficientBalance (account created with 0 balance), got %v", err)
	}

	user, err := testQueries.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("fetch user: %v", err)
	}

	bal, _ := user.Balance.Float64Value()
	if !bal.Valid || bal.Float64 != 0 {
		t.Errorf("expected balance 0 for new user, got %v", bal.Float64)
	}
}

func TestReserveBalance_Concurrent(t *testing.T) {
	ctx := context.Background()
	seed := seedTestData(ctx, t, "100.00")

	repo := ledger.NewPostgresRepository(testPool)
	const goroutines = 10
	const amount = 5.0

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.ReserveBalance(ctx, seed.UserID, decimal.NewFromFloat(amount), uuid.NewString())
			if err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent reserve error: %v", err)
	}

	user, err := testQueries.GetUserByID(ctx, seed.UserID)
	if err != nil {
		t.Fatalf("fetch user: %v", err)
	}

	bal, _ := user.Balance.Float64Value()
	if !bal.Valid {
		t.Fatal("balance is null")
	}

	expected := 100.0 - goroutines*amount
	if bal.Float64 != expected {
		t.Errorf("expected balance %.2f, got %.2f", expected, bal.Float64)
	}
}

func TestConfirmReservation(t *testing.T) {
	ctx := context.Background()
	seed := seedTestData(ctx, t, "100.00")

	repo := ledger.NewPostgresRepository(testPool)
	requestID := uuid.NewString()

	_, err := repo.ReserveBalance(ctx, seed.UserID, decimal.NewFromFloat(30), requestID)
	if err != nil {
		t.Fatalf("ReserveBalance: %v", err)
	}

	err = repo.ConfirmReservation(ctx, requestID)
	if err != nil {
		t.Fatalf("ConfirmReservation: %v", err)
	}

	user, err := testQueries.GetUserByID(ctx, seed.UserID)
	if err != nil {
		t.Fatalf("fetch user: %v", err)
	}

	bal, _ := user.Balance.Float64Value()
	if !bal.Valid || bal.Float64 != 70 {
		t.Errorf("expected balance 70 (still deducted), got %v", bal.Float64)
	}

	refUUID, _ := uuid.Parse(requestID)
	entries, err := testQueries.ListLedgerEntriesByAccount(ctx, repository.ListLedgerEntriesByAccountParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}

	var foundReservation, foundDeduction bool
	for _, e := range entries {
		if e.ReferenceID.Valid && e.ReferenceID.Bytes == refUUID {
			if e.EntryType == repository.EntryTypeReservation && e.Status == repository.LedgerStatusCompleted {
				foundReservation = true
			}
			if e.EntryType == repository.EntryTypeDeduction && e.Status == repository.LedgerStatusCompleted {
				foundDeduction = true
			}
		}
	}
	if !foundReservation {
		t.Error("expected reservation entry with status completed")
	}
	if !foundDeduction {
		t.Error("expected deduction entry with status completed")
	}
}

func TestConfirmReservation_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := ledger.NewPostgresRepository(testPool)
	err := repo.ConfirmReservation(ctx, uuid.NewString())
	if err != ledger.ErrReservationNotFound {
		t.Fatalf("expected ErrReservationNotFound, got %v", err)
	}
}

func TestConfirmReservation_NotPending(t *testing.T) {
	ctx := context.Background()
	seed := seedTestData(ctx, t, "100.00")

	repo := ledger.NewPostgresRepository(testPool)
	requestID := uuid.NewString()

	_, err := repo.ReserveBalance(ctx, seed.UserID, decimal.NewFromFloat(30), requestID)
	if err != nil {
		t.Fatalf("ReserveBalance: %v", err)
	}

	err = repo.ConfirmReservation(ctx, requestID)
	if err != nil {
		t.Fatalf("first confirm: %v", err)
	}

	err = repo.ConfirmReservation(ctx, requestID)
	if err != ledger.ErrReservationNotPending {
		t.Fatalf("expected ErrReservationNotPending on second confirm, got %v", err)
	}
}

func TestReleaseReservation(t *testing.T) {
	ctx := context.Background()
	seed := seedTestData(ctx, t, "100.00")

	repo := ledger.NewPostgresRepository(testPool)
	requestID := uuid.NewString()

	_, err := repo.ReserveBalance(ctx, seed.UserID, decimal.NewFromFloat(30), requestID)
	if err != nil {
		t.Fatalf("ReserveBalance: %v", err)
	}

	err = repo.ReleaseReservation(ctx, requestID)
	if err != nil {
		t.Fatalf("ReleaseReservation: %v", err)
	}

	user, err := testQueries.GetUserByID(ctx, seed.UserID)
	if err != nil {
		t.Fatalf("fetch user: %v", err)
	}

	bal, _ := user.Balance.Float64Value()
	if !bal.Valid || bal.Float64 != 100 {
		t.Errorf("expected balance restored to 100, got %v", bal.Float64)
	}

	refUUID, _ := uuid.Parse(requestID)
	entries, err := testQueries.ListLedgerEntriesByAccount(ctx, repository.ListLedgerEntriesByAccountParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}

	var foundCancelledReservation, foundRefund bool
	for _, e := range entries {
		if e.ReferenceID.Valid && e.ReferenceID.Bytes == refUUID {
			if e.EntryType == repository.EntryTypeReservation && e.Status == repository.LedgerStatusCancelled {
				foundCancelledReservation = true
			}
			if e.EntryType == repository.EntryTypeRefund && e.Status == repository.LedgerStatusCompleted {
				foundRefund = true
			}
		}
	}
	if !foundCancelledReservation {
		t.Error("expected cancelled reservation entry")
	}
	if !foundRefund {
		t.Error("expected refund entry")
	}
}

func TestReleaseReservation_NotFound(t *testing.T) {
	ctx := context.Background()

	repo := ledger.NewPostgresRepository(testPool)
	err := repo.ReleaseReservation(ctx, uuid.NewString())
	if err != ledger.ErrReservationNotFound {
		t.Fatalf("expected ErrReservationNotFound, got %v", err)
	}
}

func TestUserBalance(t *testing.T) {
	ctx := context.Background()
	userID := seedUserOnly(ctx, t)

	user, err := testQueries.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("fetch user: %v", err)
	}

	if user.ID != userID {
		t.Errorf("expected user id %s, got %s", userID, user.ID)
	}

	bal, _ := user.Balance.Float64Value()
	if !bal.Valid || bal.Float64 != 0 {
		t.Errorf("expected balance 0 for new user, got %v", bal.Float64)
	}
}

func TestListEntriesByAccount(t *testing.T) {
	ctx := context.Background()
	seed := seedTestData(ctx, t, "100.00")

	repo := ledger.NewPostgresRepository(testPool)

	firstID := uuid.NewString()
	secondID := uuid.NewString()

	_, err := repo.ReserveBalance(ctx, seed.UserID, decimal.NewFromFloat(10), firstID)
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}

	_, err = repo.ReserveBalance(ctx, seed.UserID, decimal.NewFromFloat(20), secondID)
	if err != nil {
		t.Fatalf("second reserve: %v", err)
	}

	entries, err := testQueries.ListLedgerEntriesByAccount(ctx, repository.ListLedgerEntriesByAccountParams{
		UserID: seed.UserID,
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListLedgerEntriesByAccount: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	firstAmt, _ := entries[0].Amount.Float64Value()
	if !firstAmt.Valid || firstAmt.Float64 != 20 {
		t.Errorf("expected first entry amount 20 (most recent), got %v", firstAmt.Float64)
	}

	secondAmt, _ := entries[1].Amount.Float64Value()
	if !secondAmt.Valid || secondAmt.Float64 != 10 {
		t.Errorf("expected second entry amount 10, got %v", secondAmt.Float64)
	}

	paginated, err := testQueries.ListLedgerEntriesByAccount(ctx, repository.ListLedgerEntriesByAccountParams{
		UserID: seed.UserID,
		Limit:  1,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("paginated list: %v", err)
	}

	if len(paginated) != 1 {
		t.Fatalf("expected 1 entry with limit 1, got %d", len(paginated))
	}

	empty, err := testQueries.ListLedgerEntriesByAccount(ctx, repository.ListLedgerEntriesByAccountParams{
		UserID: seed.UserID,
		Limit:  10,
		Offset: 10,
	})
	if err != nil {
		t.Fatalf("offset list: %v", err)
	}

	if len(empty) != 0 {
		t.Fatalf("expected 0 entries with offset 10, got %d", len(empty))
	}
}
