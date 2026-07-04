//go:build integration

package settlement

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcwait "github.com/testcontainers/testcontainers-go/wait"

	"castellan/internal/repository/db"
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

	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		return dbContainer.Terminate, fmt.Errorf("could not run migration: %w", err)
	}

	return dbContainer.Terminate, nil
}

const migrationSQL = `
CREATE TYPE currency AS ENUM ('XLM', 'USDC');
CREATE TYPE entry_type AS ENUM ('deposit', 'reservation', 'deduction', 'refund', 'settlement');
CREATE TYPE ledger_status AS ENUM ('pending', 'completed', 'failed', 'cancelled');
CREATE TYPE provider_status AS ENUM ('active', 'inactive', 'suspended');
CREATE TYPE usage_status AS ENUM ('pending', 'reserved', 'completed', 'refunded', 'failed');
CREATE TYPE batch_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE settlement_entry_status AS ENUM ('pending', 'completed', 'failed');

CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                   TEXT NOT NULL UNIQUE,
    deposit_memo            TEXT UNIQUE,
    payout_stellar_address  TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE providers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    status      provider_status NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE accounts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance     NUMERIC(20,10) NOT NULL DEFAULT 0,
    currency    currency NOT NULL DEFAULT 'XLM',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE usage_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consumer_id   UUID NOT NULL REFERENCES users(id),
    provider_id   UUID NOT NULL REFERENCES providers(id),
    endpoint_id   UUID,
    request_cost  NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    request_id    TEXT NOT NULL UNIQUE,
    status        usage_status NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

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

CREATE TABLE settlement_batches (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status        batch_status NOT NULL DEFAULT 'pending',
    total_amount  NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    entry_count   INT NOT NULL DEFAULT 0,
    tx_hash       TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at  TIMESTAMPTZ
);

CREATE TABLE settlement_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id        UUID NOT NULL REFERENCES settlement_batches(id) ON DELETE CASCADE,
    provider_id     UUID NOT NULL REFERENCES providers(id),
    amount          NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    wallet_address  TEXT NOT NULL,
    status          settlement_entry_status NOT NULL DEFAULT 'pending',
    tx_hash         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

type seedData struct {
	userID     uuid.UUID
	providerID uuid.UUID
}

func seedProviderWithUsageEvents(
	ctx context.Context, t *testing.T, email string, payoutAddr string, amounts []string,
) seedData {
	t.Helper()

	providerUserID := uuid.New()
	_, err := testPool.Exec(ctx,
		`INSERT INTO users (id, email, payout_stellar_address) VALUES ($1, $2, $3)`,
		providerUserID, email, payoutAddr)
	if err != nil {
		t.Fatalf("seed provider user: %v", err)
	}

	providerID := uuid.New()
	_, err = testPool.Exec(ctx,
		`INSERT INTO providers (id, owner_id, name, base_url) VALUES ($1, $2, $3, $4)`,
		providerID, providerUserID, "test-provider-"+email, "https://example.com")
	if err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	for i, amt := range amounts {
		consumerUserID := uuid.New()
		_, err = testPool.Exec(ctx,
			`INSERT INTO users (id, email) VALUES ($1, $2)`,
			consumerUserID, fmt.Sprintf("consumer-%s-%d@example.com", email, i))
		if err != nil {
			t.Fatalf("seed consumer user: %v", err)
		}

		requestID := uuid.New().String()
		_, err = testPool.Exec(ctx,
			`INSERT INTO usage_events (id, consumer_id, provider_id, request_cost, currency, request_id, status)
			 VALUES ($1, $2, $3, $4, 'XLM', $5, 'completed')`,
			uuid.New(), consumerUserID, providerID, amt, requestID)
		if err != nil {
			t.Fatalf("seed usage event %d: %v", i, err)
		}
	}

	return seedData{userID: providerUserID, providerID: providerID}
}

func seedSettledProvider(ctx context.Context, t *testing.T, providerID uuid.UUID, amount string) {
	t.Helper()

	batchID := uuid.New()
	_, err := testPool.Exec(ctx,
		`INSERT INTO settlement_batches (id, status, total_amount, currency, entry_count)
		 VALUES ($1, 'completed', $2, 'XLM', 1)`,
		batchID, amount)
	if err != nil {
		t.Fatalf("seed settlement batch: %v", err)
	}

	_, err = testPool.Exec(ctx,
		`INSERT INTO settlement_entries (id, batch_id, provider_id, amount, currency, wallet_address, status)
		 VALUES ($1, $2, $3, $4, 'XLM', 'GAXXXX', 'completed')`,
		uuid.New(), batchID, providerID, amount)
	if err != nil {
		t.Fatalf("seed settlement entry: %v", err)
	}
}

func TestAggregate_ReturnsUnsettledProviders(t *testing.T) {
	ctx := context.Background()

	p1 := seedProviderWithUsageEvents(ctx, t, "p1@example.com", "GAXXXX1", []string{"10.50", "5.25"})
	p2 := seedProviderWithUsageEvents(ctx, t, "p2@example.com", "GAXXXX2", []string{"20.00"})

	agg := NewAggregator(testQueries)
	payouts, err := agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}

	if len(payouts) != 2 {
		t.Fatalf("expected 2 payouts, got %d", len(payouts))
	}

	var p1Payout, p2Payout *ProviderPayout
	for i := range payouts {
		switch payouts[i].ProviderID {
		case p1.providerID:
			p1Payout = &payouts[i]
		case p2.providerID:
			p2Payout = &payouts[i]
		}
	}

	if p1Payout == nil {
		t.Fatal("expected payout for provider 1")
	}
	if !p1Payout.Amount.Equal(decimal.NewFromFloat(15.75)) {
		t.Errorf("p1 amount = %s, want 15.75", p1Payout.Amount.String())
	}
	if p1Payout.Currency != repository.CurrencyXLM {
		t.Errorf("p1 currency = %s, want XLM", p1Payout.Currency)
	}
	if p1Payout.WalletAddress != "GAXXXX1" {
		t.Errorf("p1 wallet = %s, want GAXXXX1", p1Payout.WalletAddress)
	}

	if p2Payout == nil {
		t.Fatal("expected payout for provider 2")
	}
	if !p2Payout.Amount.Equal(decimal.NewFromFloat(20.00)) {
		t.Errorf("p2 amount = %s, want 20.00", p2Payout.Amount.String())
	}
}

func TestAggregate_SkipsSettledProviders(t *testing.T) {
	ctx := context.Background()

	p1 := seedProviderWithUsageEvents(ctx, t, "settled-p1@example.com", "GAXXXX1", []string{"100.00"})
	p2 := seedProviderWithUsageEvents(ctx, t, "settled-p2@example.com", "GAXXXX2", []string{"50.00"})

	seedSettledProvider(ctx, t, p1.providerID, "100.00")

	agg := NewAggregator(testQueries)
	payouts, err := agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}

	if len(payouts) != 1 {
		t.Fatalf("expected 1 payout (p2 unsettled), got %d", len(payouts))
	}
	if payouts[0].ProviderID != p2.providerID {
		t.Errorf("expected provider %s, got %s", p2.providerID, payouts[0].ProviderID)
	}
}

func TestAggregate_SkipsZeroAmount(t *testing.T) {
	ctx := context.Background()

	_ = seedProviderWithUsageEvents(ctx, t, "zero@example.com", "GAZERO", []string{"0.00"})

	agg := NewAggregator(testQueries)
	payouts, err := agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}

	for _, p := range payouts {
		if p.Amount.Equal(decimal.Zero) || p.Amount.IsNegative() {
			t.Errorf("expected no zero/negative payouts, got %s for provider %s", p.Amount.String(), p.ProviderID)
		}
	}
}

func TestAggregate_SkipsNoWallet(t *testing.T) {
	ctx := context.Background()

	_ = seedProviderWithUsageEvents(ctx, t, "nowallet@example.com", "", []string{"50.00"})

	agg := NewAggregator(testQueries)
	payouts, err := agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}

	for _, p := range payouts {
		if p.WalletAddress == "" {
			t.Errorf("expected no payout with empty wallet, got provider %s", p.ProviderID)
		}
	}
}

func TestAggregate_Idempotent(t *testing.T) {
	ctx := context.Background()

	seedProviderWithUsageEvents(ctx, t, "idem@example.com", "GAIDEM", []string{"30.00", "20.00"})

	agg := NewAggregator(testQueries)

	first, err := agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("first Aggregate: %v", err)
	}

	second, err := agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("second Aggregate: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("first had %d payouts, second had %d", len(first), len(second))
	}

	for i := range first {
		if first[i].ProviderID != second[i].ProviderID {
			t.Errorf("mismatch provider at %d: %s vs %s", i, first[i].ProviderID, second[i].ProviderID)
		}
		if !first[i].Amount.Equal(second[i].Amount) {
			t.Errorf("mismatch amount at %d: %s vs %s", i, first[i].Amount.String(), second[i].Amount.String())
		}
	}
}

func TestAggregate_NoMutations(t *testing.T) {
	ctx := context.Background()

	seedProviderWithUsageEvents(ctx, t, "nomut@example.com", "GANOMUT", []string{"15.00"})

	var beforeUsageEvents int
	err := testPool.QueryRow(ctx, `SELECT COUNT(*) FROM usage_events`).Scan(&beforeUsageEvents)
	if err != nil {
		t.Fatalf("count usage_events before: %v", err)
	}

	var beforeSettlementBatches int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_batches`).Scan(&beforeSettlementBatches)
	if err != nil {
		t.Fatalf("count settlement_batches before: %v", err)
	}

	agg := NewAggregator(testQueries)
	_, err = agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}

	var afterUsageEvents int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM usage_events`).Scan(&afterUsageEvents)
	if err != nil {
		t.Fatalf("count usage_events after: %v", err)
	}
	if afterUsageEvents != beforeUsageEvents {
		t.Errorf("usage_events changed: before %d, after %d", beforeUsageEvents, afterUsageEvents)
	}

	var afterSettlementBatches int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_batches`).Scan(&afterSettlementBatches)
	if err != nil {
		t.Fatalf("count settlement_batches after: %v", err)
	}
	if afterSettlementBatches != beforeSettlementBatches {
		t.Errorf("settlement_batches changed: before %d, after %d", beforeSettlementBatches, afterSettlementBatches)
	}
}
