//go:build integration

package dashboard_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcwait "github.com/testcontainers/testcontainers-go/wait"

	"castellan/internal/accounts"
	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/provider"
	"castellan/internal/repository/db"
	"castellan/internal/settlement"
	"castellan/internal/usage"
)

var testPool *pgxpool.Pool
var testQueries *repository.Queries

var (
	testConsumerID  = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testProviderID  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testProvider2ID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	testEndpoint1ID = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	testEndpoint2ID = uuid.MustParse("00000000-0000-0000-0000-000000000011")
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

	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		return dbContainer.Terminate, fmt.Errorf("could not read migrations dir: %w", err)
	}

	var migrationFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			migrationFiles = append(migrationFiles, e.Name())
		}
	}
	sort.Strings(migrationFiles)

	for _, name := range migrationFiles {
		raw, err := os.ReadFile(filepath.Join("../../migrations", name))
		if err != nil {
			return dbContainer.Terminate, fmt.Errorf("could not read %s: %w", name, err)
		}
		upSQL := extractUpMigration(string(raw))
		if _, err := pool.Exec(ctx, upSQL); err != nil {
			return dbContainer.Terminate, fmt.Errorf("could not run %s: %w", name, err)
		}
	}

	return dbContainer.Terminate, nil
}

func extractUpMigration(sql string) string {
	if idx := strings.Index(sql, "-- +goose Down"); idx >= 0 {
		return sql[:idx]
	}
	return sql
}

func authenticatedRequest(t *testing.T, method, target string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      testConsumerID.String(),
			IsAuthenticated: true,
		}),
	)
	return req
}

func requireStringField(t *testing.T, obj map[string]any, field string) {
	t.Helper()
	v, ok := obj[field]
	if !ok {
		t.Errorf("response missing field %q", field)
		return
	}
	if _, ok := v.(string); !ok {
		t.Errorf("field %q = %v (type %T), want string", field, v, v)
	}
}

func TestContract_UsageEndpoint(t *testing.T) {
	ctx := context.Background()
	seedBaseData(ctx, t)
	seedUsageEvents(ctx, t)

	svc := usage.NewService(testQueries)
	h := usage.NewHandler(svc)

	t.Run("returns paginated shape with string amounts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ListUsage(rec, authenticatedRequest(t, "GET", "/api/v1/usage?limit=50"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		data, ok := resp["data"].([]any)
		if !ok {
			t.Fatal("data is not an array")
		}
		if len(data) == 0 {
			t.Fatal("expected at least one usage event")
		}
		item, ok := data[0].(map[string]any)
		if !ok {
			t.Fatal("data[0] is not an object")
		}

		requireStringField(t, item, "request_cost")
		requireStringField(t, item, "id")
		requireStringField(t, item, "route")
		requireStringField(t, item, "method")
		requireStringField(t, item, "currency")

		if next, ok := resp["next_cursor"]; !ok {
			t.Error("missing next_cursor")
		} else if next != nil {
			if _, ok := next.(string); !ok {
				t.Errorf("next_cursor = %v (type %T), want string or null", next, next)
			}
		}
	})

	t.Run("cursor pagination across pages", func(t *testing.T) {
		rec1 := httptest.NewRecorder()
		h.ListUsage(rec1, authenticatedRequest(t, "GET", "/api/v1/usage?limit=2"))
		if rec1.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec1.Code, rec1.Body.String())
		}
		var page1 map[string]any
		if err := json.Unmarshal(rec1.Body.Bytes(), &page1); err != nil {
			t.Fatalf("decode page1: %v", err)
		}
		data1 := page1["data"].([]any)
		if len(data1) != 2 {
			t.Fatalf("expected 2 items on page, got %d", len(data1))
		}
		next := page1["next_cursor"]
		if next == nil {
			t.Fatal("expected next_cursor for paginated response")
		}
		cursor, ok := next.(string)
		if !ok {
			t.Fatalf("next_cursor type = %T, want string", next)
		}

		rec2 := httptest.NewRecorder()
		req2 := authenticatedRequest(t, "GET", "/api/v1/usage?limit=2&cursor="+cursor)
		h.ListUsage(rec2, req2)
		if rec2.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
		}
		var page2 map[string]any
		if err := json.Unmarshal(rec2.Body.Bytes(), &page2); err != nil {
			t.Fatalf("decode page2: %v", err)
		}
		data2 := page2["data"].([]any)
		if len(data2) == 0 {
			t.Fatal("expected items on page 2")
		}
	})
}

func TestContract_SettlementsEndpoint(t *testing.T) {
	ctx := context.Background()
	seedBaseData(ctx, t)
	seedSettlements(ctx, t)

	reconciler := settlement.NewReconciler(testPool, testQueries)
	h := settlement.NewHandler(reconciler)

	// The handler passes ConsumerID directly as the provider UUID to the SQL query.
	// Use the provider UUID (not the user UUID) so the JOIN matches.
	providerUUID := testProvider2ID
	req := httptest.NewRequest("GET", "/api/v1/settlements", nil)
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      providerUUID.String(),
			IsAuthenticated: true,
		}),
	)

	t.Run("returns nested entries with string amounts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ListSettlements(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		data, ok := resp["data"].([]any)
		if !ok {
			t.Fatal("data is not an array")
		}
		if len(data) == 0 {
			t.Fatal("expected at least one settlement batch")
		}
		batch, ok := data[0].(map[string]any)
		if !ok {
			t.Fatal("data[0] not an object")
		}

		requireStringField(t, batch, "total_amount")
		requireStringField(t, batch, "id")
		requireStringField(t, batch, "status")

		if _, ok := batch["entry_count"].(float64); !ok {
			t.Errorf("entry_count = %v (type %T), want float64", batch["entry_count"], batch["entry_count"])
		}

		entries, ok := batch["entries"].([]any)
		if !ok {
			t.Fatal("entries is not an array")
		}
		if len(entries) == 0 {
			t.Fatal("expected at least one settlement entry")
		}
		entry, ok := entries[0].(map[string]any)
		if !ok {
			t.Fatal("entries[0] not an object")
		}
		requireStringField(t, entry, "amount")
		requireStringField(t, entry, "provider_id")
		requireStringField(t, entry, "wallet_address")
	})
}

func TestContract_ProvidersEndpoint(t *testing.T) {
	ctx := context.Background()
	seedBaseData(ctx, t)

	svc := provider.NewProviderService(testQueries)
	h := provider.NewProviderHandler(svc)

	t.Run("returns raw JSON array", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ListProviders(rec, authenticatedRequest(t, "GET", "/api/v1/providers"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		// Response is a raw array at root level
		var providers []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &providers); err != nil {
			t.Fatalf("expected raw JSON array, got error: %v. Body: %s", err, rec.Body.String())
		}
		if len(providers) == 0 {
			t.Fatal("expected at least one provider")
		}
		p := providers[0]
		requireStringField(t, p, "id")
		requireStringField(t, p, "name")
		requireStringField(t, p, "base_url")
		requireStringField(t, p, "status")
		requireStringField(t, p, "owner_id")
	})
}

func TestContract_BalanceEndpoint(t *testing.T) {
	ctx := context.Background()
	seedBaseData(ctx, t)

	svc := accounts.NewService(testQueries)
	h := accounts.NewHandler(svc)

	t.Run("returns string amounts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.GetBalance(rec, authenticatedRequest(t, "GET", "/api/v1/balance"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		requireStringField(t, resp, "balance")
		requireStringField(t, resp, "available_balance")
		requireStringField(t, resp, "currency")
	})
}

func TestContract_AccountEndpoint(t *testing.T) {
	ctx := context.Background()
	seedBaseData(ctx, t)

	svc := accounts.NewService(testQueries)
	h := accounts.NewHandler(svc)

	t.Run("returns balance as string", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.GetAccount(rec, authenticatedRequest(t, "GET", "/api/v1/accounts/me"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		requireStringField(t, resp, "balance")
		requireStringField(t, resp, "id")
		requireStringField(t, resp, "currency")
		requireStringField(t, resp, "created_at")
	})
}

func TestContract_AccountEntriesEndpoint(t *testing.T) {
	ctx := context.Background()
	seedBaseData(ctx, t)
	seedLedgerEntries(ctx, t)

	svc := accounts.NewService(testQueries)
	h := accounts.NewHandler(svc)

	t.Run("returns entries with string amounts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ListEntries(rec, authenticatedRequest(t, "GET", "/api/v1/accounts/me/entries?limit=50"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		entries, ok := resp["entries"].([]any)
		if !ok {
			t.Fatal("entries is not an array")
		}
		if len(entries) == 0 {
			t.Fatal("expected at least one ledger entry")
		}
		entry, ok := entries[0].(map[string]any)
		if !ok {
			t.Fatal("entries[0] not an object")
		}
		requireStringField(t, entry, "amount")
		requireStringField(t, entry, "balance_after")
		requireStringField(t, entry, "entry_type")
		requireStringField(t, entry, "status")

		if _, ok := resp["total"].(float64); !ok {
			t.Errorf("total = %v (type %T), want float64", resp["total"], resp["total"])
		}
		if _, ok := resp["limit"].(float64); !ok {
			t.Errorf("limit = %v (type %T), want float64", resp["limit"], resp["limit"])
		}
		if _, ok := resp["offset"].(float64); !ok {
			t.Errorf("offset = %v (type %T), want float64", resp["offset"], resp["offset"])
		}
	})
}

func TestContract_EarningsEndpoint(t *testing.T) {
	ctx := context.Background()
	seedBaseData(ctx, t)
	seedUsageEvents(ctx, t)

	svc := usage.NewService(testQueries)
	h := usage.NewHandler(svc)

	t.Run("returns string amounts", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.GetEarnings(rec, authenticatedRequest(t, "GET", "/api/v1/earnings"))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		requireStringField(t, resp, "total_earnings")
		requireStringField(t, resp, "unsettled_earnings")

		byEndpoint, ok := resp["by_endpoint"].([]any)
		if !ok {
			t.Fatal("by_endpoint is not an array")
		}
		if len(byEndpoint) > 0 {
			ep, ok := byEndpoint[0].(map[string]any)
			if !ok {
				t.Fatal("by_endpoint[0] not an object")
			}
			requireStringField(t, ep, "total")
		}

		sparkline, ok := resp["sparkline"].([]any)
		if !ok {
			t.Fatal("sparkline is not an array")
		}
		if len(sparkline) > 0 {
			sl, ok := sparkline[0].(map[string]any)
			if !ok {
				t.Fatal("sparkline[0] not an object")
			}
			requireStringField(t, sl, "amount")
		}
	})
}

// ── Seed helpers ──

func seedBaseData(ctx context.Context, t *testing.T) {
	t.Helper()
	cleanDB(ctx, t)

	// Users
	for _, u := range []struct {
		id    uuid.UUID
		email string
	}{
		{testConsumerID, "consumer@test.com"},
		{testProviderID, "provider-owner@test.com"},
	} {
		_, err := testPool.Exec(ctx,
			`INSERT INTO users (id, email) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`,
			u.id, u.email)
		if err != nil {
			t.Fatalf("seed user %s: %v", u.id, err)
		}
	}

	// Account for consumer with known balance
	_, err := testPool.Exec(ctx,
		`INSERT INTO accounts (id, owner_id, balance, currency)
		 VALUES ($1, $2, 1000.00, 'XLM')
		 ON CONFLICT (owner_id) DO UPDATE SET balance = 1000.00`,
		uuid.MustParse("00000000-0000-0000-0000-000000000100"), testConsumerID)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}

	// Providers
	for i, p := range []struct {
		id      uuid.UUID
		owner   uuid.UUID
		name    string
		baseURL string
	}{
		{testProvider2ID, testConsumerID, "weather-api", "https://api.weather.example.com"},
		{uuid.MustParse("00000000-0000-0000-0000-000000000004"), testProviderID, "ai-inference", "https://inference.ai.example.com"},
	} {
		_, err := testPool.Exec(ctx,
			`INSERT INTO providers (id, owner_id, name, base_url, status)
			 VALUES ($1, $2, $3, $4, 'active')
			 ON CONFLICT (id) DO NOTHING`,
			p.id, p.owner, p.name, p.baseURL)
		if err != nil {
			t.Fatalf("seed provider %d: %v", i, err)
		}
	}

	// Endpoints
	for _, e := range []struct {
		id         uuid.UUID
		providerID uuid.UUID
		route      string
		method     string
		price      string
	}{
		{testEndpoint1ID, testProvider2ID, "/v1/weather/current", "GET", "0.01"},
		{testEndpoint2ID, testProvider2ID, "/v1/weather/forecast", "GET", "0.02"},
	} {
		_, err := testPool.Exec(ctx,
			`INSERT INTO api_endpoints (id, provider_id, route, method, price_amount, currency)
			 VALUES ($1, $2, $3, $4, $5, 'XLM')
			 ON CONFLICT (id) DO NOTHING`,
			e.id, e.providerID, e.route, e.method, e.price)
		if err != nil {
			t.Fatalf("seed endpoint %s: %v", e.route, err)
		}
	}
}

func seedUsageEvents(ctx context.Context, t *testing.T) {
	t.Helper()
	now := time.Now().UTC()

	for i := range 5 {
		ts := now.Add(-time.Duration(i) * time.Hour)
		_, err := testPool.Exec(ctx,
			`INSERT INTO usage_events (id, consumer_id, provider_id, endpoint_id, request_cost, currency, status_code, latency_ms, response_size, request_id, status, created_at)
			 VALUES ($1, $2, $3, $4, $5, 'XLM', $6, $7, $8, $9, 'completed', $10)
			 ON CONFLICT (request_id) DO NOTHING`,
			uuid.New(), testConsumerID, testProvider2ID, testEndpoint1ID, "0.01",
			200, 150, 1024, fmt.Sprintf("req-contract-usage-%d", i), ts)
		if err != nil {
			t.Fatalf("seed usage event %d: %v", i, err)
		}
	}
}

func seedLedgerEntries(ctx context.Context, t *testing.T) {
	t.Helper()
	accountID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	now := time.Now().UTC()

	// Deposit
	_, err := testPool.Exec(ctx,
		`INSERT INTO ledger_entries (id, account_id, entry_type, amount, balance_after, currency, status, created_at)
		 VALUES ($1, $2, 'deposit', 1000.00, 1000.00, 'XLM', 'completed', $3)
		 ON CONFLICT (id) DO NOTHING`,
		uuid.MustParse("00000000-0000-0000-0000-000000000200"), accountID, now.Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("seed deposit entry: %v", err)
	}

	// Deduction
	_, err = testPool.Exec(ctx,
		`INSERT INTO ledger_entries (id, account_id, entry_type, amount, balance_after, currency, status, created_at)
		 VALUES ($1, $2, 'deduction', -0.05, 999.95, 'XLM', 'completed', $3)
		 ON CONFLICT (id) DO NOTHING`,
		uuid.MustParse("00000000-0000-0000-0000-000000000201"), accountID, now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("seed deduction entry: %v", err)
	}
}

func seedSettlements(ctx context.Context, t *testing.T) {
	t.Helper()
	now := time.Now().UTC()

	batchID := uuid.MustParse("00000000-0000-0000-0000-000000000300")
	_, err := testPool.Exec(ctx,
		`INSERT INTO settlement_batches (id, status, total_amount, currency, entry_count, created_at)
		 VALUES ($1, 'completed', 25.00, 'XLM', 1, $2)
		 ON CONFLICT (id) DO NOTHING`,
		batchID, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("seed settlement batch: %v", err)
	}

	_, err = testPool.Exec(ctx,
		`INSERT INTO settlement_entries (id, batch_id, provider_id, amount, currency, wallet_address, status, created_at)
		 VALUES ($1, $2, $3, 15.00, 'XLM', 'GABCDEF123', 'completed', $4)
		 ON CONFLICT (id) DO NOTHING`,
		uuid.MustParse("00000000-0000-0000-0000-000000000301"), batchID, testProvider2ID, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("seed settlement entry: %v", err)
	}
}

func cleanDB(ctx context.Context, t *testing.T) {
	t.Helper()
	tables := []string{
		"settlement_entries", "settlement_batches", "deposits",
		"usage_events", "ledger_entries", "accounts",
		"api_endpoints", "providers", "api_keys", "users",
	}
	for _, table := range tables {
		if _, err := testPool.Exec(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("clean table %s: %v", table, err)
		}
	}
}
