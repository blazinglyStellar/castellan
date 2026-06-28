# Castellan — Agent Instructions

## Structure
- `cmd/api/main.go` — Go server entrypoint (net/http + ServeMux, no framework)
- `dashboard/` — Next.js 15 (App Router, shadcn/ui, Tailwind)
- `internal/database/` — pgx via `database/sql`, singleton, testcontainers for integration tests
- `internal/server/` — routes, handlers, std lib mux
- `internal/repository/` — **does not exist yet**; sqlc not run. Run `sqlc generate` to create it.
- `migrations/` — goose SQL migrations (`000001_init.sql` exists, 10 tables, 10 enums)

## Env
- `.env` autoloaded by `godotenv/autoload` in both `server.go` and `database.go` — no manual loading
- **Gotcha**: Go code reads `BLUEPRINT_DB_*` env vars (`BLUEPRINT_DB_DATABASE`, `BLUEPRINT_DB_USERNAME`, etc.) but `.env.example` uses `DB_*` prefix. If you add new env vars, update `.env.example` and `docker-compose.yml` to match the actual prefix the code reads.
- `DATABASE_URL` / `GOOSE_DBSTRING` in `.env.example` are for goose CLI only — the app constructs its own connection string from `BLUEPRINT_DB_*`.
- `PORT` for HTTP (default 8080)

## Backend Commands
- `make build` — `go build -o main.exe cmd/api/main.go` (Windows `.exe`); Dockerfile builds Linux binary to `/bin/castellan`
- `make run` — `go run cmd/api/main.go`
- `make test` — `go test ./... -v`
- `make itest` — `go test -tags=integration ./internal/provider/... ./internal/database/... ./internal/gateway/... -v` (requires Docker, spins up testcontainers Postgres)
- `make test` — `go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./... -v` (skips integration-tagged tests; use `make itest` for those)
- `make watch` — air live-reload (auto-installs if missing, Windows PowerShell)
- `make docker-run` / `make docker-down` — Docker Compose for DB infra
- Full suite with race: `go test -race -count=1 ./...` (matches CI)

## Dashboard Commands
- `npm run dev` — Next.js dev server (port 3000)
- `npm run build` / `npm run lint`

## Linting
- `golangci-lint run ./...` — requires v2.12+ (config is v2 format; CI pins `GOLANGCI_LINT_VERSION: v2.12`)
- gofumpt enforces 3-group import layout: stdlib / project (`castellan/...`) / third-party
- sloglint `attr-only`: use `slog.String()`, `slog.Int()`, etc., never raw k/v pairs
- sloglint `key-naming-case: snake`, `context: scope`, `msg-style: lowercased`
- Reserved slog keys forbidden: `time`, `level`, `msg`, `source`
- Context must propagate through call chain — `contextcheck` and revive `context-as-argument` are enforced
- No `panic` outside `main` — return errors
- Test files (`_test.go`) are exempted from `errcheck`, `gosec`, `noctx`
- `cmd/api/main.go` exempted from `gosec` (hardcoded credentials) and `mnd` (magic numbers)
- `internal/server/server.go` and `internal/server/providers.go` line `Magic number: 8` exempted from `mnd`

## Database & Codegen
- sqlc: schema from `migrations/`, queries from `internal/repository/query/`, output to `internal/repository/db/` (sqlc.yaml)
- sqlc uses `pgx/v5`, generates `emit_interface: true`, `emit_json_tags: true`, UUID → `google/uuid`, numeric → `shopspring/decimal`
- goose: `DATABASE_URL="postgres://castellan:castellan@localhost:5432/castellan?sslmode=disable" goose -s -dir migrations postgres "$DATABASE_URL" up`
- Integration tests (database package) require Docker

## Stale References
- `.github/workflows/integration-testing.yml` still uses `POSTGRES_DB: flowgate` and `DB_NAME: flowgate` — needs rename
- `.github/workflows/trivy.yml` still uses `IMAGE_NAME: flowgate`
