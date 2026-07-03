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
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

var (
	testPG      *pgx.Conn
	testQueries *repository.Queries
	testRdb     *goredis.Client
)

func TestMain(m *testing.M) {
	pgTeardown, err := mustStartPostgresContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not start postgres container: %v\n", err)
		os.Exit(1)
	}

	redisTeardown, err := mustStartRedisContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not start redis container: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if testPG != nil {
		if err := testPG.Close(context.Background()); err != nil {
			slog.Warn("could not close database connection", slog.String("error", err.Error()))
		}
	}

	if pgTeardown != nil {
		if err := pgTeardown(context.Background()); err != nil {
			slog.Warn("could not teardown postgres container", slog.String("error", err.Error()))
		}
	}

	if redisTeardown != nil {
		if err := redisTeardown(context.Background()); err != nil {
			slog.Warn("could not teardown redis container", slog.String("error", err.Error()))
		}
	}

	os.Exit(code)
}

func mustStartRedisContainer() (func(context.Context, ...testcontainers.TerminateOption) error, error) {
	ctx := context.Background()

	container, err := tcredis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			tcwait.ForLog("* Ready to accept connections").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("redis container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		return container.Terminate, fmt.Errorf("connection string: %w", err)
	}

	opts, err := goredis.ParseURL(connStr)
	if err != nil {
		return container.Terminate, fmt.Errorf("parse URL: %w", err)
	}

	testRdb = goredis.NewClient(opts)

	return container.Terminate, nil
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

CREATE TYPE session_token_status AS ENUM ('active', 'revoked', 'expired');

CREATE TABLE session_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    label       TEXT,
    scope       TEXT,
    status      session_token_status NOT NULL DEFAULT 'active',
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_session_tokens_user ON session_tokens (user_id);
CREATE INDEX idx_session_tokens_status ON session_tokens (status);
`

const testRawKey = "ca_test-key-for-integration"

type seedData struct {
	UserID     uuid.UUID
	APIKeyID   uuid.UUID
	ProviderID uuid.UUID
	EndpointID uuid.UUID
	AccountID  uuid.UUID
}

func seedGatewayTestData(ctx context.Context, t *testing.T, baseURL, balance, price string, rateLimit ...int) seedData {
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
	var rl *int
	if len(rateLimit) > 0 && rateLimit[0] > 0 {
		rl = &rateLimit[0]
	}
	_, err = testPG.Exec(ctx,
		`INSERT INTO api_endpoints (id, provider_id, route, method, price_amount, currency, rate_limit, status) VALUES ($1, $2, $3, $4, $5, 'XLM', $6, 'active')`,
		endpointID, providerID, "/v1/chat", "POST", price, rl)
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
			var rateLimit *int
			err = conn.QueryRow(r.Context(),
				`SELECT id, price_amount::text, currency::text, rate_limit FROM api_endpoints
				 WHERE provider_id = $1 AND route = $2 AND status = 'active'
				 LIMIT 1`,
				providerUUID, route,
			).Scan(&endpointID, &priceAmount, &currency, &rateLimit)
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
			maxRequests := 0
			if rateLimit != nil {
				maxRequests = *rateLimit
			}
			ctx = gatewaycontext.SetRateLimitInfo(ctx, gatewaycontext.RateLimitInfo{
				MaxRequests:   maxRequests,
				WindowSeconds: 60,
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
	rateLimiter ...gateway.RateLimiter,
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

	sessionSvc := auth.NewSessionService(testQueries)

	var h http.Handler = pxy
	h = middleware.UsageCapture(usageRepo, nil)(h)
	h = middleware.Reservation(ledger)(h)
	h = middleware.BalanceCheck(balanceChecker)(h)
	h = middleware.MaxBodySize(10 * 1024 * 1024)(h)
	if len(rateLimiter) > 0 && rateLimiter[0] != nil {
		h = middleware.RateLimitCheck(rateLimiter[0])(h)
	}
	h = pricingMiddleware(conn)(h)
	h = middleware.AuthCheck(
		middleware.KeyValidatorFunc(testQueries.GetKeyByHash),
		middleware.SessionValidatorFunc(sessionSvc.ValidateSessionToken),
	)(h)

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

// seedSessionToken creates a fresh session token for the given user and returns
// the raw st_ token string. duration controls how long until it expires.
func seedSessionToken(ctx context.Context, t *testing.T, userID uuid.UUID, duration time.Duration) string {
	t.Helper()

	svc := auth.NewSessionService(testQueries)
	rawToken, _, err := svc.GenerateSessionToken(ctx, userID, "test-session", nil, duration)
	if err != nil {
		t.Fatalf("seed session token: %v", err)
	}

	return rawToken
}

func TestGatewayAPIKeyRevoked(t *testing.T) {
	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, "http://0.0.0.0:1", "1000.00", "5.00")

	// Give this test an isolated key hash so no other seeded row (which all share
	// the testRawKey hash) can satisfy the lookup after this key is revoked.
	revokedRawKey := "ca_revoked-key-" + seed.APIKeyID.String()
	_, err := testPG.Exec(ctx,
		`UPDATE api_keys SET key_hash = $1, status = 'revoked' WHERE id = $2`,
		auth.HashKey(revokedRawKey), seed.APIKeyID)
	if err != nil {
		t.Fatalf("revoke api key: %v", err)
	}

	tracker := newLedgerTracker()
	handler := buildGatewayHandler(testPG, tracker, proxy.DefaultConfig())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+revokedRawKey)
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
	if body["error"] != "api key revoked" {
		t.Fatalf("expected error 'api key revoked'; got %q", body["error"])
	}

	if len(tracker.Calls) != 0 {
		t.Fatalf("expected 0 ledger calls on revoked key; got %d", len(tracker.Calls))
	}

	events, err := testQueries.ListUsageEventsByConsumer(ctx, seed.UserID)
	if err != nil {
		t.Fatalf("failed to query usage events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 usage events for revoked key; got %d", len(events))
	}
}

func TestGatewayAPIKeyExpired(t *testing.T) {
	ctx := context.Background()

	// Create a fresh user so the expired key has a unique hash.
	userID := uuid.New()
	email := fmt.Sprintf("expired-%s@example.com", userID.String())
	_, err := testPG.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, userID, email)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	accountID := uuid.New()
	_, err = testPG.Exec(ctx,
		`INSERT INTO accounts (id, owner_id, balance) VALUES ($1, $2, $3)`,
		accountID, userID, "1000.00")
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}

	providerID := uuid.New()
	_, err = testPG.Exec(ctx,
		`INSERT INTO providers (id, owner_id, name, base_url, status) VALUES ($1, $2, $3, $4, 'active')`,
		providerID, userID, "expired-key-provider", "http://0.0.0.0:1")
	if err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	_, err = testPG.Exec(ctx,
		`INSERT INTO api_endpoints (id, provider_id, route, method, price_amount, currency, status) VALUES ($1, $2, $3, $4, $5, 'XLM', 'active')`,
		uuid.New(), providerID, "/v1/chat", "POST", "5.00")
	if err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}

	// Insert an API key that is already past its expiry.
	expiredRawKey := "ca_expired-key-lifecycle-test"
	keyHash := auth.HashKey(expiredRawKey)
	_, err = testPG.Exec(ctx,
		`INSERT INTO api_keys (id, user_id, key_hash, status, expires_at) VALUES ($1, $2, $3, 'active', NOW() - INTERVAL '1 hour')`,
		uuid.New(), userID, keyHash)
	if err != nil {
		t.Fatalf("seed expired api key: %v", err)
	}

	tracker := newLedgerTracker()
	handler := buildGatewayHandler(testPG, tracker, proxy.DefaultConfig())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", providerID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+expiredRawKey)
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
	if body["error"] != "api key expired" {
		t.Fatalf("expected error 'api key expired'; got %q", body["error"])
	}

	if len(tracker.Calls) != 0 {
		t.Fatalf("expected 0 ledger calls on expired key; got %d", len(tracker.Calls))
	}

	events, err := testQueries.ListUsageEventsByConsumer(ctx, userID)
	if err != nil {
		t.Fatalf("failed to query usage events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 usage events for expired key; got %d", len(events))
	}
}

func TestGatewaySessionTokenValid(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer mockUpstream.Close()

	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, mockUpstream.URL, "1000.00", "5.00")
	rawToken := seedSessionToken(ctx, t, seed.UserID, 1*time.Hour)

	ledger := newLedgerTracker()
	handler := buildGatewayHandler(testPG, ledger, proxy.DefaultConfig())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+rawToken)
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
}

func TestGatewaySessionTokenRevoked(t *testing.T) {
	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, "http://0.0.0.0:1", "1000.00", "5.00")
	rawToken := seedSessionToken(ctx, t, seed.UserID, 1*time.Hour)

	// Look up the token record so we can revoke it by ID.
	tokenHash := auth.HashToken(rawToken)
	tokenRecord, err := testQueries.GetSessionTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("get session token by hash: %v", err)
	}

	if _, err := testQueries.RevokeSessionToken(ctx, tokenRecord.ID); err != nil {
		t.Fatalf("revoke session token: %v", err)
	}

	tracker := newLedgerTracker()
	handler := buildGatewayHandler(testPG, tracker, proxy.DefaultConfig())

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+rawToken)
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
	if body["error"] != "session token revoked" {
		t.Fatalf("expected error 'session token revoked'; got %q", body["error"])
	}

	if len(tracker.Calls) != 0 {
		t.Fatalf("expected 0 ledger calls on revoked session; got %d", len(tracker.Calls))
	}

	events, err := testQueries.ListUsageEventsByConsumer(ctx, seed.UserID)
	if err != nil {
		t.Fatalf("failed to query usage events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 usage events for revoked session; got %d", len(events))
	}
}

func TestGatewayRateLimit_WithinLimit(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer mockUpstream.Close()

	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, mockUpstream.URL, "1000.00", "5.00", 10)

	ledger := newLedgerTracker()
	rateLimiter := gateway.NewRedisRateLimiter(testRdb, 60)
	handler := buildGatewayHandler(testPG, ledger, proxy.DefaultConfig(), rateLimiter)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost,
			fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
			strings.NewReader(`{"prompt":"hello"}`))
		req.Header.Set("Authorization", "Bearer "+testRawKey)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200; got %d, body: %s", i+1, rec.Code, rec.Body.String())
		}

		if v := rec.Header().Get("X-Ratelimit-Limit"); v != "10" {
			t.Errorf("request %d: expected X-Ratelimit-Limit 10, got %q", i+1, v)
		}
		if v := rec.Header().Get("X-Ratelimit-Remaining"); v == "" {
			t.Errorf("request %d: expected X-Ratelimit-Remaining header", i+1)
		}
		if v := rec.Header().Get("X-Ratelimit-Reset"); v == "" {
			t.Errorf("request %d: expected X-Ratelimit-Reset header", i+1)
		}
	}
}

func TestGatewayRateLimit_Exceeded(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer mockUpstream.Close()

	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, mockUpstream.URL, "1000.00", "5.00", 10)

	ledger := newLedgerTracker()
	rateLimiter := gateway.NewRedisRateLimiter(testRdb, 60)
	handler := buildGatewayHandler(testPG, ledger, proxy.DefaultConfig(), rateLimiter)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost,
			fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
			strings.NewReader(`{"prompt":"hello"}`))
		req.Header.Set("Authorization", "Bearer "+testRawKey)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200; got %d, body: %s", i+1, rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
		strings.NewReader(`{"prompt":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+testRawKey)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429; got %d, body: %s", rec.Code, rec.Body.String())
	}

	if v := rec.Header().Get("Retry-After"); v == "" {
		t.Error("expected Retry-After header on 429")
	}
	if v := rec.Header().Get("X-Ratelimit-Limit"); v != "10" {
		t.Errorf("expected X-Ratelimit-Limit 10, got %q", v)
	}
	if v := rec.Header().Get("X-Ratelimit-Remaining"); v != "0" {
		t.Errorf("expected X-Ratelimit-Remaining 0, got %q", v)
	}
	if v := rec.Header().Get("X-Ratelimit-Reset"); v == "" {
		t.Error("expected X-Ratelimit-Reset header")
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON body: %v", err)
	}
	if body["error"] != "rate_limit_exceeded" {
		t.Errorf("expected error 'rate_limit_exceeded', got %q", body["error"])
	}
}

func TestGatewayRateLimit_UnlimitedEndpoint(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer mockUpstream.Close()

	ctx := context.Background()
	seed := seedGatewayTestData(ctx, t, mockUpstream.URL, "1000.00", "5.00")

	ledger := newLedgerTracker()
	rateLimiter := gateway.NewRedisRateLimiter(testRdb, 60)
	handler := buildGatewayHandler(testPG, ledger, proxy.DefaultConfig(), rateLimiter)

	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodPost,
			fmt.Sprintf("/api/gateway/%s/v1/chat", seed.ProviderID.String()),
			strings.NewReader(`{"prompt":"hello"}`))
		req.Header.Set("Authorization", "Bearer "+testRawKey)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200; got %d, body: %s", i+1, rec.Code, rec.Body.String())
		}
	}
}

func TestGatewayRateLimit_ConsumerIsolation(t *testing.T) {
	mockUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer mockUpstream.Close()

	ctx := context.Background()

	// Consumer A — owns provider+endpoint
	rawKeyA := "ca_rate-limit-isolation-a"
	seedA := seedGatewayTestData(ctx, t, mockUpstream.URL, "1000.00", "5.00", 1)
	_, err := testPG.Exec(ctx,
		`UPDATE api_keys SET key_hash = $1 WHERE user_id = $2`,
		auth.HashKey(rawKeyA), seedA.UserID)
	if err != nil {
		t.Fatalf("update api key A: %v", err)
	}

	// Consumer B — different user, uses the same provider+endpoint
	userBID := uuid.New()
	_, err = testPG.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`,
		userBID, fmt.Sprintf("isolation-b-%s@example.com", userBID.String()))
	if err != nil {
		t.Fatalf("seed user B: %v", err)
	}
	rawKeyB := "ca_rate-limit-isolation-b"
	keyHashB := auth.HashKey(rawKeyB)
	_, err = testPG.Exec(ctx,
		`INSERT INTO api_keys (id, user_id, key_hash, status) VALUES ($1, $2, $3, 'active')`,
		uuid.New(), userBID, keyHashB)
	if err != nil {
		t.Fatalf("seed api key B: %v", err)
	}
	_, err = testPG.Exec(ctx,
		`INSERT INTO accounts (id, owner_id, balance) VALUES ($1, $2, $3)`,
		uuid.New(), userBID, "1000.00")
	if err != nil {
		t.Fatalf("seed account B: %v", err)
	}

	ledger := newLedgerTracker()
	rateLimiter := gateway.NewRedisRateLimiter(testRdb, 60)
	handler := buildGatewayHandler(testPG, ledger, proxy.DefaultConfig(), rateLimiter)

	doRequest := func(key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost,
			fmt.Sprintf("/api/gateway/%s/v1/chat", seedA.ProviderID.String()),
			strings.NewReader(`{"prompt":"hello"}`))
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	rec := doRequest(rawKeyA)
	if rec.Code != http.StatusOK {
		t.Fatalf("consumer A request 1: expected 200; got %d, body: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(rawKeyB)
	if rec.Code != http.StatusOK {
		t.Fatalf("consumer B request 1: expected 200; got %d, body: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(rawKeyA)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("consumer A request 2: expected 429; got %d, body: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(rawKeyB)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("consumer B request 2: expected 429; got %d, body: %s", rec.Code, rec.Body.String())
	}
}
