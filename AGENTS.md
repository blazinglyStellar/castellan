# Castellan

Usage-based API monetization gateway (Stellar settlement, PostgreSQL ledger, Redis rate limiting planned but not wired).

## Structure

| Path | What |
|---|---|
| `cmd/api/main.go` | Go server entrypoint (std lib `net/http` + `ServeMux` — no Chi, no framework) |
| `dashboard/` | Next.js 15 App Router, shadcn/ui, Tailwind |
| `internal/server/` | `NewServer()`, `RegisterRoutes()`, middleware chain |
| `internal/server/middleware/` | BalanceCheck → Reservation → UsageCapture → Proxy (pipeline order) |
| `internal/proxy/` | `httputil.ReverseProxy` + jittered retry round tripper |
| `internal/database/` | pgxpool singleton, `DB_*` env vars |
| `internal/repository/db/` | **sqlc-generated** (package `repository`), checked in |
| `internal/repository/query/` | sqlc source `.sql` files |
| `internal/provider/` | `DBResolver` — resolves upstream base URL |
| `internal/gateway/` | `LedgerService` interface |
| `migrations/` | Goose SQL migrations (10 tables, 10 enums in `000001_init.sql`) |
| `internal/gateway/context/` | Request-scoped context values (consumer, pricing, metrics) |

## Commands

### Backend
```
make lint          — golangci-lint v2.12+ (config is v2 format)
make test          — go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./... -v
make itest         — go test -v -tags=integration ./internal/provider/... ./internal/database/... ./internal/gateway/...
make build / run   — go build -o main.exe cmd/api/main.go / go run cmd/api/main.go
make ci / ci-full  — lint → vet → test → security / +itest +trivy
make docker-run    — docker compose up --build (psql only, no Redis)
go test -race -count=1 ./...   — full suite matching CI
```

### Dashboard
```
npm run dev        — Next.js on :3000
npm run build/lint — production build / lint
```

### Codegen
```
sqlc generate      — schema: migrations/ | queries: internal/repository/query/ | output: internal/repository/db/
```
sqlc: pgx/v5, emit_interface+json_tags, UUID→google/uuid, numeric→shopspring/decimal.

### Migration
```
DATABASE_URL="postgres://castellan:castellan@localhost:5432/castellan?sslmode=disable" \
  goose -s -dir migrations postgres "$DATABASE_URL" up
```

## Gateway Request Lifecycle

```
Request → RequestID → RequestLogger → Recovery → CORS
  → MaxBodySize → BalanceCheck → Reservation → UsageCapture → Proxy
```

Middleware uses `func(http.Handler) http.Handler`. Function adapters (`BalanceCheckerFunc`, `UsageEventRepositoryFunc`) bridge interfaces. Post-response ledger ops use `context.WithoutCancel` to survive client disconnect.

## Gotchas

- **`PORT` has no default fallback.** `strconv.Atoi(os.Getenv("PORT"))` — if unset/empty, Atoi returns 0 → random port. `.env.example` sets `PORT=8080` but the app doesn't default to it.
- **CI integration-testing.yml is broken:** runs `./integration/...` — that directory doesn't exist; Makefile's paths are correct.
- **`internal/database/database_test.go`** uses testcontainers but has NO `//go:build integration` tag, unlike every other testcontainers test. `make test` will try Docker.
- **NoopLedger** — deprecated, removed in #102.
- **No auth middleware** — API key auth documented but not wired. Integration tests inject mock auth.
- **No Redis** — docker-compose has no Redis service; no rate-limit code exists.
- **sqlc filename typo:** `internal/repository/db/getAccountBalaance.sql.go` (double 'a'). Function `GetAccountBalance` is correct.
- **Integration tests embed SQL inline** — they don't apply the real migration file.

## Linting (non-default rules)

- gofumpt: 3-group imports: stdlib / `castellan/...` / third-party
- sloglint: `attr-only` (use `slog.String()`, never raw pairs), `key-naming-case: snake`, `context: scope`, `msg-style: lowercased`
- Reserved slog keys: `time`, `level`, `msg`, `source` — forbidden
- `contextcheck` + revive `context-as-argument`: context must propagate
- No `panic` outside `main`
- Test files exempt: `errcheck`, `gosec`, `noctx`, `revive`, `wrapcheck`

## Stale in docs but not code

- README mentions "Chi router" — code uses std lib `ServeMux` (Go 1.22+ pattern matching)
- `.github/workflows/` have already been renamed from `flowgate` to `castellan`; docs saying otherwise are outdated

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, invoke the `skill` tool with `skill: "graphify"` before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
