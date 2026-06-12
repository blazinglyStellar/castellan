# Castellan — Agent Instructions

## Structure
- `cmd/api/main.go` — Go server entrypoint (net/http + ServeMux, no framework)
- `dashboard/` — Next.js 15 (App Router, shadcn/ui, Tailwind)
- `internal/database/` — pgx via `database/sql`, testcontainers for integration tests
- `internal/server/` — routes, handlers, std lib mux
- `internal/repository/query/` — sqlc query source files
- `internal/repository/db/` — sqlc generated Go (checked in)

## Env
- `.env` autoloaded by godotenv/autoload — no manual loading
- DB env vars use `BLUEPRINT_DB_*` prefix (`BLUEPRINT_DB_DATABASE`, `BLUEPRINT_DB_USERNAME`, etc.)
- `PORT` for HTTP (default 8080)

## Backend Commands (Makefile)
- `make build` — `go build -o main.exe cmd/api/main.go`
- `make run` — `go run cmd/api/main.go`
- `make test` — `go test ./... -v`
- `make itest` — `go test ./internal/database -v` (requires Docker, spins up testcontainers Postgres)
- `make watch` — air live-reload (auto-installs if missing)
- `make docker-run` / `make docker-down` — Docker Compose for DB infra

## Dashboard Commands
- `npm run dev` — Next.js dev server (port 3000)
- `npm run build` / `npm run lint`

## Linting
- `golangci-lint run ./...` — requires v2.12+ (v1.x config format differs)
- gofumpt enforces 3-group import layout: stdlib / project(`castellan/...`) / third-party
- sloglint `attr-only` — use `slog.String()`, `slog.Int()`, etc., never raw k/v pairs
- Context must propagate through call chain (no `context.Background()` outside main)
- No `panic` outside main — return errors

## Database & Codegen
- sqlc: schema from `migrations/`, queries from `internal/repository/query/`, output to `internal/repository/db/`
- goose for migrations: `DATABASE_URL="postgres://castellan:castellan@localhost:5432/castellan?sslmode=disable" goose -s -dir migrations postgres "$DATABASE_URL" up`
- Integration tests (database package) require Docker (testcontainers spins up Postgres)
- No migrations files exist yet; no sqlc queries exist yet

## Testing
- `make test` runs all tests (`go test ./... -v`)
- `make itest` runs database integration tests only (Docker required, slow)
- Route tests use `httptest.NewServer` (no external deps)
- `go test -race -count=1 ./...` for full suite with race detection
