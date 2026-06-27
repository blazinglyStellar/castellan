//go:build integration

package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
CREATE TYPE provider_status AS ENUM ('active', 'inactive', 'suspended');

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
	status      provider_status NOT NULL DEFAULT 'active',
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func TestDBResolver_ResolveBaseURL_ActiveProvider(t *testing.T) {
	queries := repository.New(testDB)
	resolver, err := NewDBResolver(queries)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}
	ctx := context.Background()

	userID := uuid.New()
	_, err := testDB.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, userID, "active-test@example.com")
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	providerID := uuid.New()
	expectedURL := "https://api.active-provider.com"
	_, err = testDB.Exec(ctx,
		`INSERT INTO providers (id, owner_id, name, base_url, status) VALUES ($1, $2, $3, $4, 'active')`,
		providerID, userID, "active-provider", expectedURL)
	if err != nil {
		t.Fatalf("failed to seed provider: %v", err)
	}

	got, err := resolver.ResolveBaseURL(ctx, providerID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expectedURL {
		t.Fatalf("expected %q, got %q", expectedURL, got)
	}
}

func TestDBResolver_ResolveBaseURL_InactiveProvider(t *testing.T) {
	queries := repository.New(testDB)
	resolver, err := NewDBResolver(queries)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}
	ctx := context.Background()

	userID := uuid.New()
	_, err := testDB.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, userID, "inactive-test@example.com")
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	providerID := uuid.New()
	_, err = testDB.Exec(ctx,
		`INSERT INTO providers (id, owner_id, name, base_url, status) VALUES ($1, $2, $3, $4, 'inactive')`,
		providerID, userID, "inactive-provider", "https://api.inactive-provider.com")
	if err != nil {
		t.Fatalf("failed to seed provider: %v", err)
	}

	_, err = resolver.ResolveBaseURL(ctx, providerID.String())
	if err == nil {
		t.Fatal("expected error for inactive provider, got nil")
	}
}
