# Castellan

Usage-based API monetization gateway (Stellar settlement, PostgreSQL ledger, Redis rate limiting).

## Structure

| Path | What |
|---|---|
| `cmd/api/main.go` | Go server entrypoint (std lib `net/http` + `ServeMux` — no Chi, no framework) |
| `cmd/worker/main.go` | Background worker (deposit watcher + settlement runner) |
| `cmd/seed/main.go` | Database seed command |
| `dashboard-v3/` | Next.js 15 App Router, shadcn/ui, Tailwind |
| `internal/server/` | `NewServer()`, `RegisterRoutes()`, middleware chain |
| `internal/server/middleware/` | 8 middleware layers (see lifecycle below) |
| `internal/proxy/` | `httputil.ReverseProxy` + jittered retry round tripper |
| `internal/database/` | pgxpool singleton, `DB_*` env vars |
| `internal/repository/db/` | **sqlc-generated** (package `repository`), checked in |
| `internal/repository/query/` | sqlc source `.sql` files |
| `internal/provider/` | `DBResolver` + provider/endpoint CRUD services & handlers |
| `internal/gateway/` | `LedgerService` interface, `RedisRateLimiter` |
| `internal/gateway/context/` | Request-scoped context values (consumer, pricing, metrics) |
| `internal/auth/` | Key/session management, OAuth (Google/GitHub via goth) |
| `internal/accounts/` | Account handler + service |
| `internal/ledger/` | `PostgresLedger` implementation |
| `internal/settlement/` | Aggregator, Stellar submitter, reconciler, cycle runner |
| `internal/stellar/` | Stellar config, Horizon client helpers |
| `migrations/` | Goose SQL migrations |
| `internal/database/database_test.go` | testcontainers-based, **no** `//go:build integration` tag |

## Commands

### Backend
```
make lint          — golangci-lint v2 (config has top-level formatters: key)
make test          — go test -race -count=1 ./...  (will try Docker from database_test.go)
make itest         — go test -v -tags=integration ./internal/provider/... ./internal/database/... ./internal/gateway/... ./internal/ledger/... ./internal/settlement/...
make build / run   — cp docs/openapi.yaml internal/server/openapi.yaml; go build/run cmd/api/main.go
make ci            — lint → vet → test → build
make ci-full       — ci + itest + trivy-scan
make docker-run    — clean → build-linux → docker compose up -d (postgres + redis + api + settlement-worker)
make seed          — go run cmd/seed/main.go  (sample providers/endpoints)
make db-reset      — db-drop → migrate → seed
make build-linux   — cross-compile both binaries to bin/ for linux/amd64
go test -race -count=1 ./...  — full suite (will try Docker)
```

### Dashboard
```
npm run dev        — Next.js on :3000
npm run build/lint — production build / lint
npm test           — vitest run (not jest)
```

### Codegen
```
sqlc generate      — schema: migrations/ | queries: internal/repository/query/ | output: internal/repository/db/
```
sqlc: pgx/v5, emit_interface+json_tags, UUID→google/uuid, numeric→shopspring/decimal.

### Migration
```
make migrate       — goose up (reads .env for DB_* vars)
make migrate-down  — goose down
make db-drop       — psql DROP/CREATE DATABASE
```

## Gateway Request Lifecycle

Applied via `RegisterRoutes()` then `GatewayRoutes()` — wrapping is inside-out, so the actual request order is:

```
[Global] Recovery → RequestLogger → RequestID → CORS
  [Gateway only, outermost first]
  SetRequestStart → AuthCheck (API key `ca_` or session `st_` or cookie)
    → PricingResolver (endpoint lookup + rate limit config)
    → RateLimitCheck (Redis token bucket)
    → BalanceCheck (402 if insufficient)
    → MaxBodySize
    → UsageCapture (records event on response)
    → Reservation (reserve funds, then commit/release post-response)
    → Proxy (httputil.ReverseProxy + jittered retry)
```

Middleware uses `func(http.Handler) http.Handler`. Function adapters (`BalanceCheckerFunc`, `UsageEventRepositoryFunc`) bridge interfaces. Post-response ledger ops use `context.WithoutCancel` to survive client disconnect.

## Linting (non-default rules)

- gofumpt: 3-group imports — stdlib / `castellan/...` / third-party
- sloglint: `attr-only` (use `slog.String()`, never raw pairs), `key-naming-case: snake`, `context: scope`, `msg-style: lowercased`
- Reserved slog keys: `time`, `level`, `msg`, `source` — forbidden
- `contextcheck` + revive `context-as-argument`: context must propagate
- `revive` with `enable-all-rules: true` — ~60 rules, many at error severity
- `varnamelen` with `min-name-length: 1` (effectively off)
- `goconst` with `min-occurrences: 10`
- Test files exempt: `errcheck`, `gosec`, `noctx`, `revive`, `wrapcheck`

## Gotchas

- **`PORT` has no default fallback.** `strconv.Atoi(os.Getenv("PORT"))` — if unset/empty, Atoi returns 0 → random port. `.env.example` sets `PORT=8080`.
- **`database_test.go` uses testcontainers but lacks `//go:build integration`.** `make test` will try Docker. Unlike every other integration test.
- **Makefile `include .env`** — runs `include .env` + `export`, then reconstructs `DATABASE_URL` from individual `DB_*` vars. Keep `.env` present.
- **`SEED=true` env var** triggers seed inside `cmd/api/main.go` before the server starts. `make seed` runs `cmd/seed/main.go` independently.
- **sqlc filename typo:** `internal/repository/db/getAccountBalaance.sql.go` (double 'a'). Function `GetAccountBalance` is correct.
- **Integration tests embed SQL inline** — they don't apply the real migration file.
- **README says "Chi router" and "zap logging"** — code uses std lib `ServeMux` and `log/slog`. Both are stale.
- **README says `dashboard/`** — the directory is `dashboard-v3/`.
- **CI workflows directory is empty.** `.github/workflows/` has no files; CI targets in Makefile may not match any running CI.
- **No Redis in docker-compose.redis** — never mind, it IS in docker-compose.yml. Redis is present.
- **OAuth requires env vars** — `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `SESSION_STORE_SECRET`, `DASHBOARD_URL`. Without these, the auth endpoints will fail.
- **`STELLAR_HOT_WALLET_ADDRESS` is required** — `NewServer()` returns an error if unset.

## graphify

This project has a knowledge graph at graphify-out/. When the user types `/graphify`, invoke the `skill` tool with `skill: "graphify"` before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
