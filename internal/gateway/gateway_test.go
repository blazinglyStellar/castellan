//go:build integration

package gateway_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"castellan/internal/auth"
	"castellan/internal/gateway"
	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/provider"
	"castellan/internal/proxy"
	"castellan/internal/repository/db"
	"castellan/internal/server/middleware"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testPG      *pgx.Conn
	testQueries *repository.Queries
)

func TestMain(m *testing.M) {
	teardown, err := mustStartPostgresContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not start postgres container: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if testPG != nil {
		if err := testPG.Close(context.Background()); err != nil {
			slog.Warn("could not close database connection", slog.String("error", err.Error()))
		}
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

	testPG = conn
	testQueries = repository.New(conn)

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

CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                   TEXT NOT NULL UNIQUE,
    deposit_memo            TEXT UNIQUE,
    payout_stellar_address  TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash    TEXT NOT NULL,
    label       TEXT,
    status      api_key_status NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ
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

const testRawKey = "ca_test-key-for-integration"

type seedData struct {
	UserID     uuid.UUID
	APIKeyID   uuid.UUID
	ProviderID uuid.UUID
	EndpointID uuid.UUID
	AccountID  uuid.UUID
}

func seedGatewayTestData(ctx context.Context, t *testing.T, baseURL, balance, price string) seedData {
	t.Helper()

	userID := uuid.New()
	email := fmt.Sprintf("test-%s@example.com", userID.String())
	_, err := testPG.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, userID, email)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	apiKeyID := uuid.New()
	keyHash := auth.HashKey(testRawKey)
	_, err = testPG.Exec(ctx,
		`INSERT INTO api_keys (id, user_id, key_hash, status) VALUES ($1, $2, $3, 'active')`,
		apiKeyID, userID, keyHash)
	if err != nil {
		t.Fatalf("seed api_key: %v", err)
	}

	accountID := uuid.New()
	_, err = testPG.Exec(ctx,
		`INSERT INTO accounts (id, owner_id, balance) VALUES ($1, $2, $3)`,
		accountID, userID, balance)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}

	providerID := uuid.New()
	_, err = testPG.Exec(ctx,
		`INSERT INTO providers (id, owner_id, name, base_url, status) VALUES ($1, $2, $3, $4, 'active')`,
		providerID, userID, "test-provider", baseURL)
	if err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	endpointID := uuid.New()
	_, err = testPG.Exec(ctx,
		`INSERT INTO api_endpoints (id, provider_id, route, method, price_amount, currency, status) VALUES ($1, $2, $3, $4, $5, 'XLM', 'active')`,
		endpointID, providerID, "/v1/chat", "POST", price)
	if err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}

	return seedData{
		UserID:     userID,
		APIKeyID:   apiKeyID,
		ProviderID: providerID,
		EndpointID: endpointID,
		AccountID:  accountID,
	}
}

type ledgerCall struct {
	Method      string
	ReferenceID string
}

type mockLedgerService struct {
	mu    sync.Mutex
	Calls []ledgerCall
}

func newLedgerTracker() *mockLedgerService {
	return &mockLedgerService{}
}

func (m *mockLedgerService) Reserve(_ context.Context, _ uuid.UUID, _ decimal.Decimal, referenceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, ledgerCall{Method: "Reserve", ReferenceID: referenceID})
	return nil
}

func (m *mockLedgerService) Commit(_ context.Context, referenceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, ledgerCall{Method: "Commit", ReferenceID: referenceID})
	return nil
}

func (m *mockLedgerService) Release(_ context.Context, referenceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, ledgerCall{Method: "Release", ReferenceID: referenceID})
	return nil
}

func pricingMiddleware(conn *pgx.Conn) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providerID, route, ok := parseGatewayPath(r.URL.Path)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"})
				return
			}

			providerUUID, err := uuid.Parse(providerID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid provider id"})
				return
			}

			var endpointID uuid.UUID
			var priceAmount string
			var currency string
			err = conn.QueryRow(r.Context(),
				`SELECT id, price_amount::text, currency::text FROM api_endpoints
				 WHERE provider_id = $1 AND route = $2 AND status = 'active'
				 LIMIT 1`,
				providerUUID, route,
			).Scan(&endpointID, &priceAmount, &currency)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "endpoint not found"})
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "pricing lookup failed"})
				return
			}

			priceDec, err := decimal.NewFromString(priceAmount)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid price"})
				return
			}

			ctx := gatewaycontext.SetPricingInfo(r.Context(), gatewaycontext.PricingInfo{
				EndpointID:  endpointID.String(),
				ProviderID:  providerUUID.String(),
				PriceAmount: priceDec,
				Currency:    gatewaycontext.Currency(currency),
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseGatewayPath(path string) (providerID, rest string, ok bool) {
	const prefix = "/api/gateway/"
	const splitParts = 2

	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(trimmed, "/", splitParts)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}

	providerID = parts[0]

	if len(parts) > 1 {
		rest = "/" + parts[1]
	} else {
		rest = "/"
	}

	return providerID, rest, true
}

func buildGatewayHandler(
	conn *pgx.Conn,
	ledger gateway.LedgerService,
	proxyCfg proxy.Config,
) http.Handler {
	resolver, err := provider.NewDBResolver(testQueries)
	if err != nil {
		panic(fmt.Sprintf("buildGatewayHandler: %v", err))
	}

	balanceChecker := middleware.BalanceCheckerFunc(func(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error) {
		balance, err := testQueries.GetAccountBalance(ctx, ownerID)
		if err != nil {
			return decimal.Zero, fmt.Errorf("balance unavailable: %w", err)
		}
		f64, err := balance.Float64Value()
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to convert balance: %w", err)
		}
		if !f64.Valid {
			return decimal.Zero, errors.New("balance is null")
		}
		return decimal.NewFromFloat(f64.Float64), nil
	})

	usageRepo := middleware.UsageEventRepositoryFunc(func(ctx context.Context, arg repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error) {
		return testQueries.CreateUsageEvent(ctx, arg)
	})

	pxy := proxy.NewReverseProxy(resolver, slog.Default(), proxyCfg)

	var h http.Handler = pxy
	h = middleware.UsageCapture(usageRepo, nil)(h)
	h = middleware.Reservation(ledger)(h)
	h = middleware.BalanceCheck(balanceChecker)(h)
	h = middleware.MaxBodySize(10 * 1024 * 1024)(h)
	h = pricingMiddleware(conn)(h)
	h = middleware.AuthCheck(middleware.KeyValidatorFunc(testQueries.GetKeyByHash))(h)

	return h
}

func TestGatewayLifecycle(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer mockUpstream.Close()

	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, mockUpstream.URL, "1000.00", "5.00")

	ledger := newLedgerTracker()
	handler := buildGatewayHandler(testPG, ledger, proxy.DefaultConfig())

	body := strings.NewReader(`{"prompt":"hello"}`)
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		body)
	req.Header.Set("Authorization", "Bearer "+testRawKey)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200; got %d, body: %s", rec.Code, rec.Body.String())
	}

	var respBody map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}
	if respBody["message"] != "ok" {
		t.Fatalf("expected upstream response message 'ok'; got %q", respBody["message"])
	}

	if len(ledger.Calls) != 2 {
		t.Fatalf("expected 2 ledger calls; got %d: %+v", len(ledger.Calls), ledger.Calls)
	}
	if ledger.Calls[0].Method != "Reserve" {
		t.Fatalf("expected first ledger call to be Reserve; got %s", ledger.Calls[0].Method)
	}
	if ledger.Calls[1].Method != "Commit" {
		t.Fatalf("expected second ledger call to be Commit; got %s", ledger.Calls[1].Method)
	}

	events, err := testQueries.ListUsageEventsByConsumer(ctx, seed.UserID)
	if err != nil {
		t.Fatalf("failed to query usage events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 usage event; got %d", len(events))
	}
	if events[0].ConsumerID != seed.UserID {
		t.Fatalf("expected consumer_id %s; got %s", seed.UserID, events[0].ConsumerID)
	}
	if events[0].ProviderID != seed.ProviderID {
		t.Fatalf("expected provider_id %s; got %s", seed.ProviderID, events[0].ProviderID)
	}
	if events[0].EndpointID != seed.EndpointID {
		t.Fatalf("expected endpoint_id %s; got %s", seed.EndpointID, events[0].EndpointID)
	}
}

func TestGatewayAuthFailure(t *testing.T) {
	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, "http://0.0.0.0:1", "0", "5.00")

	tracker := newLedgerTracker()
	handler := buildGatewayHandler(testPG, tracker, proxy.DefaultConfig())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401; got %d, body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON error response: %v", err)
	}
	if body["error"] != "missing authorization header" {
		t.Fatalf("expected error 'missing authorization header'; got %q", body["error"])
	}

	events, err := testQueries.ListUsageEventsByConsumer(ctx, seed.UserID)
	if err != nil {
		t.Fatalf("failed to query usage events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 usage events for failed auth; got %d", len(events))
	}

	if len(tracker.Calls) != 0 {
		t.Fatalf("expected 0 ledger calls on auth failure; got %d", len(tracker.Calls))
	}
}

func TestGatewayInsufficientBalance(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not have been called for insufficient balance")
		w.WriteHeader(http.StatusOK)
	}))
	defer mockUpstream.Close()

	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, mockUpstream.URL, "0.01", "5.00")

	ledger := newLedgerTracker()
	handler := buildGatewayHandler(testPG, ledger, proxy.DefaultConfig())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+testRawKey)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expected status 402; got %d, body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON error response: %v", err)
	}
	if body["error"] != "insufficient_balance" {
		t.Fatalf("expected error 'insufficient_balance'; got %q", body["error"])
	}

	events, err := testQueries.ListUsageEventsByConsumer(ctx, seed.UserID)
	if err != nil {
		t.Fatalf("failed to query usage events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 usage events for insufficient balance; got %d", len(events))
	}

	if len(ledger.Calls) != 0 {
		t.Fatalf("expected 0 ledger calls on insufficient balance; got %d", len(ledger.Calls))
	}
}

func TestGatewayUpstreamTimeout(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer mockUpstream.Close()

	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, mockUpstream.URL, "1000.00", "5.00")

	proxyCfg := proxy.Config{
		ConnectTimeout:  1 * time.Second,
		ReadTimeout:     500 * time.Millisecond,
		WriteTimeout:    1 * time.Second,
		RetryMaxRetries: 1,
		RetryBaseDelay:  100 * time.Millisecond,
		RetryMaxDelay:   500 * time.Millisecond,
	}

	ledger := newLedgerTracker()
	handler := buildGatewayHandler(testPG, ledger, proxyCfg)

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+testRawKey)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status 504; got %d, body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON error response: %v", err)
	}
	if body["error"] != "upstream request timed out" {
		t.Fatalf("expected error 'upstream request timed out'; got %q", body["error"])
	}

	if len(ledger.Calls) != 2 {
		t.Fatalf("expected 2 ledger calls; got %d: %+v", len(ledger.Calls), ledger.Calls)
	}
	if ledger.Calls[0].Method != "Reserve" {
		t.Fatalf("expected first ledger call to be Reserve; got %s", ledger.Calls[0].Method)
	}
	if ledger.Calls[1].Method != "Release" {
		t.Fatalf("expected second ledger call to be Release; got %s", ledger.Calls[1].Method)
	}
}

func TestGatewayMissingAuthHeader(t *testing.T) {
	seed := seedGatewayTestData(context.Background(), t, "http://0.0.0.0:1", "0", "5.00")

	tracker := newLedgerTracker()
	handler := buildGatewayHandler(testPG, tracker, proxy.DefaultConfig())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401; got %d, body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON error response: %v", err)
	}
	if body["error"] != "missing authorization header" {
		t.Fatalf("expected error 'missing authorization header'; got %q", body["error"])
	}

	if len(tracker.Calls) != 0 {
		t.Fatalf("expected 0 ledger calls on missing auth; got %d", len(tracker.Calls))
	}
}

func TestGatewayInvalidBearerToken(t *testing.T) {
	seed := seedGatewayTestData(context.Background(), t, "http://0.0.0.0:1", "0", "5.00")

	tracker := newLedgerTracker()
	handler := buildGatewayHandler(testPG, tracker, proxy.DefaultConfig())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Authorization", "Bearer ca_nonexistent-key")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401; got %d, body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON error response: %v", err)
	}
	if body["error"] != "invalid api key" {
		t.Fatalf("expected error 'invalid api key'; got %q", body["error"])
	}

	if len(tracker.Calls) != 0 {
		t.Fatalf("expected 0 ledger calls on invalid token; got %d", len(tracker.Calls))
	}
}

func TestGatewayWrongBearerPrefix(t *testing.T) {
	seed := seedGatewayTestData(context.Background(), t, "http://0.0.0.0:1", "0", "5.00")

	tracker := newLedgerTracker()
	handler := buildGatewayHandler(testPG, tracker, proxy.DefaultConfig())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Authorization", "Bearer xyz_wrong-prefix")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401; got %d, body: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON error response: %v", err)
	}
	if body["error"] != "invalid api key" {
		t.Fatalf("expected error 'invalid api key'; got %q", body["error"])
	}

	if len(tracker.Calls) != 0 {
		t.Fatalf("expected 0 ledger calls on wrong prefix; got %d", len(tracker.Calls))
	}
}
