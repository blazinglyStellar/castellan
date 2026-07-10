//go:build integration

package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool

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
		dbContainer.Terminate(ctx)
		return nil, fmt.Errorf("could not get host: %w", err)
	}

	port, err := dbContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		dbContainer.Terminate(ctx)
		return nil, fmt.Errorf("could not get port: %w", err)
	}

	connStr := fmt.Sprintf("postgres://test:test@%s:%s/castellan_test?sslmode=disable", host, port.Port())

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		dbContainer.Terminate(ctx)
		return nil, fmt.Errorf("could not create connection pool: %w", err)
	}

	testPool = pool

	if _, err := pool.Exec(ctx, migrationSQL); err != nil {
		pool.Close()
		dbContainer.Terminate(ctx)
		testPool = nil
		return nil, fmt.Errorf("could not run migration: %w", err)
	}

	return dbContainer.Terminate, nil
}

const migrationSQL = `
CREATE TYPE provider_status AS ENUM ('active', 'inactive', 'suspended');
CREATE TYPE endpoint_status AS ENUM ('active', 'inactive', 'draft');
CREATE TYPE currency AS ENUM ('XLM', 'USDC');

CREATE TABLE users (
	id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	email       TEXT NOT NULL UNIQUE,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE providers (
	id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name        TEXT NOT NULL,
	base_url    TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
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
	description     TEXT NOT NULL DEFAULT '',
	status          endpoint_status NOT NULL DEFAULT 'active',
	created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
	CONSTRAINT unique_provider_route_method UNIQUE (provider_id, route, method)
);

CREATE TYPE usage_status AS ENUM ('pending', 'reserved', 'completed', 'refunded', 'failed');

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
