//go:build integration

package repository

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testDB *pgx.Conn

func TestMain(m *testing.M) {
	teardown, err := mustStartPostgresContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not start postgres container: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if testDB != nil {
		testDB.Close(context.Background())
	}

	if teardown != nil {
		if err := teardown(context.Background()); err != nil {
			log.Printf("could not teardown postgres container: %v", err)
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

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return dbContainer.Terminate, fmt.Errorf("could not connect to postgres: %w", err)
	}

	testDB = conn

	if _, err := conn.Exec(ctx, migrationSQL); err != nil {
		return dbContainer.Terminate, fmt.Errorf("could not run migration: %w", err)
	}

	return dbContainer.Terminate, nil
}

const migrationSQL = `
CREATE TYPE api_key_status AS ENUM ('active', 'revoked', 'expired');
CREATE TYPE provider_status AS ENUM ('active', 'inactive', 'suspended');
CREATE TYPE endpoint_status AS ENUM ('active', 'inactive', 'draft');
CREATE TYPE currency AS ENUM ('XLM', 'USDC');
CREATE TYPE entry_type AS ENUM ('deposit', 'reservation', 'deduction', 'refund', 'settlement');
CREATE TYPE ledger_status AS ENUM ('pending', 'completed', 'failed', 'cancelled');
CREATE TYPE usage_status AS ENUM ('pending', 'reserved', 'completed', 'refunded', 'failed');
CREATE TYPE batch_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE settlement_entry_status AS ENUM ('pending', 'completed', 'failed');
CREATE TYPE deposit_status AS ENUM ('pending', 'confirmed', 'failed');

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

CREATE TABLE api_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id     UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    route           TEXT NOT NULL,
    method          TEXT NOT NULL DEFAULT 'GET',
    price_amount    NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    rate_limit      INT,
    status          endpoint_status NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unique_provider_route_method UNIQUE (provider_id, route, method)
);

CREATE TABLE usage_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consumer_id   UUID NOT NULL REFERENCES users(id),
    provider_id   UUID NOT NULL REFERENCES providers(id),
    endpoint_id   UUID NOT NULL REFERENCES api_endpoints(id),
    request_cost  NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    status_code   INT,
    latency_ms    INT,
    response_size INT,
    request_id    TEXT NOT NULL UNIQUE,
    status        usage_status NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func newTxQueries(t *testing.T) (*Queries, context.Context, pgx.Tx) {
	t.Helper()
	ctx := context.Background()
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	t.Cleanup(func() {
		tx.Rollback(ctx)
	})
	return New(tx), ctx, tx
}

func seedTestData(ctx context.Context, t *testing.T, tx pgx.Tx) (userID, providerID, endpointID uuid.UUID) {
	t.Helper()

	userID = uuid.New()
	_, err := tx.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, userID, "consumer@test.com")
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	providerID = uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO providers (id, owner_id, name, base_url, status) VALUES ($1, $2, $3, $4, 'active')`,
		providerID, userID, "test-provider", "https://api.test.com")
	if err != nil {
		t.Fatalf("failed to seed provider: %v", err)
	}

	endpointID = uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO api_endpoints (id, provider_id, route, method, price_amount) VALUES ($1, $2, $3, $4, $5)`,
		endpointID, providerID, "/v1/test", "GET", `0.01`)
	if err != nil {
		t.Fatalf("failed to seed endpoint: %v", err)
	}

	return userID, providerID, endpointID
}

func defaultUsageParams(consumerID, providerID, endpointID uuid.UUID) CreateUsageEventParams {
	requestCost := pgtype.Numeric{}
	requestCost.Scan("0.01")

	return CreateUsageEventParams{
		ConsumerID:  consumerID,
		ProviderID:  providerID,
		EndpointID:  endpointID,
		RequestCost: requestCost,
		Currency:    CurrencyXLM,
		StatusCode:  pgtype.Int4{Int32: 200, Valid: true},
		LatencyMs:   pgtype.Int4{Int32: 42, Valid: true},
		RequestID:   uuid.New().String(),
		Status:      UsageStatusCompleted,
	}
}

func TestCreateUsageEvent(t *testing.T) {
	q, ctx, tx := newTxQueries(t)
	consumerID, providerID, endpointID := seedTestData(ctx, t, tx)

	params := defaultUsageParams(consumerID, providerID, endpointID)

	event, err := q.CreateUsageEvent(ctx, params)
	if err != nil {
		t.Fatalf("CreateUsageEvent() returned error: %v", err)
	}

	if event.ID == uuid.Nil {
		t.Fatal("expected non-zero UUID for created event")
	}
	if event.ConsumerID != consumerID {
		t.Fatalf("expected consumer_id %v, got %v", consumerID, event.ConsumerID)
	}
	if event.ProviderID != providerID {
		t.Fatalf("expected provider_id %v, got %v", providerID, event.ProviderID)
	}
	if event.EndpointID != endpointID {
		t.Fatalf("expected endpoint_id %v, got %v", endpointID, event.EndpointID)
	}
	if event.RequestID != params.RequestID {
		t.Fatalf("expected request_id %q, got %q", params.RequestID, event.RequestID)
	}
	if event.Status != UsageStatusCompleted {
		t.Fatalf("expected status %q, got %q", UsageStatusCompleted, event.Status)
	}
	if event.CreatedAt.IsZero() {
		t.Fatal("expected non-zero created_at")
	}
}

func TestCreateUsageEvent_DuplicateRequestID(t *testing.T) {
	q, ctx, tx := newTxQueries(t)
	consumerID, providerID, endpointID := seedTestData(ctx, t, tx)

	params := defaultUsageParams(consumerID, providerID, endpointID)
	requestID := uuid.New().String()
	params.RequestID = requestID

	event, err := q.CreateUsageEvent(ctx, params)
	if err != nil {
		t.Fatalf("first CreateUsageEvent() returned error: %v", err)
	}
	if event.RequestID != requestID {
		t.Fatalf("expected request_id %q, got %q", requestID, event.RequestID)
	}

	dup, err := q.CreateUsageEvent(ctx, params)
	if err != nil {
		t.Fatalf("duplicate CreateUsageEvent() returned error: %v", err)
	}
	if dup.ID != event.ID {
		t.Fatalf("expected duplicate to return same event ID %v, got %v", event.ID, dup.ID)
	}

	var count int
	tx.QueryRow(ctx, `SELECT COUNT(*) FROM usage_events`).Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 usage event, got %d", count)
	}
}

func TestGetUsageEventByRequestID(t *testing.T) {
	q, ctx, tx := newTxQueries(t)
	consumerID, providerID, endpointID := seedTestData(ctx, t, tx)

	params := defaultUsageParams(consumerID, providerID, endpointID)

	created, err := q.CreateUsageEvent(ctx, params)
	if err != nil {
		t.Fatalf("CreateUsageEvent() returned error: %v", err)
	}

	got, err := q.GetUsageEventByRequestID(ctx, params.RequestID)
	if err != nil {
		t.Fatalf("GetUsageEventByRequestID() returned error: %v", err)
	}

	if got.ID != created.ID {
		t.Fatalf("expected event ID %v, got %v", created.ID, got.ID)
	}
	if got.RequestID != params.RequestID {
		t.Fatalf("expected request_id %q, got %q", params.RequestID, got.RequestID)
	}
}

func TestGetUsageEventByRequestID_NotFound(t *testing.T) {
	q, ctx, _ := newTxQueries(t)

	_, err := q.GetUsageEventByRequestID(ctx, "non-existent-request-id")
	if err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}
}

func TestListUsageEventsByConsumer(t *testing.T) {
	q, ctx, tx := newTxQueries(t)
	consumerA, providerID, endpointID := seedTestData(ctx, t, tx)

	consumerB := uuid.New()
	_, err := tx.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, consumerB, "consumer-b@test.com")
	if err != nil {
		t.Fatalf("failed to seed consumer B: %v", err)
	}

	paramsA1 := defaultUsageParams(consumerA, providerID, endpointID)
	paramsA1.RequestID = uuid.New().String()

	paramsA2 := defaultUsageParams(consumerA, providerID, endpointID)
	paramsA2.RequestID = uuid.New().String()

	paramsB := defaultUsageParams(consumerB, providerID, endpointID)
	paramsB.RequestID = uuid.New().String()

	if _, err := q.CreateUsageEvent(ctx, paramsA1); err != nil {
		t.Fatalf("failed to create event A1: %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, err := q.CreateUsageEvent(ctx, paramsA2); err != nil {
		t.Fatalf("failed to create event A2: %v", err)
	}
	if _, err := q.CreateUsageEvent(ctx, paramsB); err != nil {
		t.Fatalf("failed to create event B: %v", err)
	}

	events, err := q.ListUsageEventsByConsumer(ctx, consumerA)
	if err != nil {
		t.Fatalf("ListUsageEventsByConsumer() returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events for consumer A, got %d", len(events))
	}

	if events[0].CreatedAt.Before(events[1].CreatedAt) {
		t.Fatal("expected events in DESC order by created_at")
	}

	eventsB, err := q.ListUsageEventsByConsumer(ctx, consumerB)
	if err != nil {
		t.Fatalf("ListUsageEventsByConsumer() for consumer B returned error: %v", err)
	}
	if len(eventsB) != 1 {
		t.Fatalf("expected 1 event for consumer B, got %d", len(eventsB))
	}

	eventsEmpty, err := q.ListUsageEventsByConsumer(ctx, uuid.New())
	if err != nil {
		t.Fatalf("ListUsageEventsByConsumer() for unknown consumer returned error: %v", err)
	}
	if len(eventsEmpty) != 0 {
		t.Fatalf("expected 0 events for unknown consumer, got %d", len(eventsEmpty))
	}
}
