# Simple Makefile for a Go project

# Default: run all CI checks, then build
all: ci build

build:
	@echo "Building..."
	@go build -o main.exe cmd/api/main.go

# Aggregate CI checks (runs before build via `all`)
ci: lint vet test build
	@echo "All CI checks passed"

ci-full: ci itest trivy-scan
	@echo "All CI checks (including integration + trivy) passed"

# Run golangci-lint (mirrors lint.yml)
lint:
	@echo "Linting..."
	@golangci-lint run ./...

# Run go vet
vet:
	@echo "Vetting..."
	@go vet ./...

# Run the application
run:
	@go run cmd/api/main.go

# Print docs URL
docs:
	@echo "OpenAPI spec: docs/openapi.yaml"
	@echo "Scalar UI:   http://localhost:${PORT}/docs"

# Lint the OpenAPI spec with redocly (requires node)
validate-docs:
	npx --yes @redocly/cli lint docs/openapi.yaml

# Cross-compile both binaries for linux/amd64 (Docker runtime target)
build-linux:
	@echo "Building linux binaries..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/castellan-api ./cmd/api/
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/castellan-worker ./cmd/worker/

# Create DB container (builds binaries on host first — no Docker OOM)
docker-run: clean build-linux
	@docker compose up --build -d

# Seed the database with sample data
seed:
	@echo "Seeding database..."
	@go run cmd/seed/main.go

# Build the worker binary
build-worker:
	@go build -o worker.exe cmd/worker/main.go

# Run the worker process
run-worker:
	@go run cmd/worker/main.go

# Shutdown DB container
docker-down:
	@docker compose down

# Apply goose migrations to localhost:5432 (override DATABASE_URL for custom hosts)
DB_USER ?= postgres
DB_PASSWORD ?= 1234
DB_DATABASE ?= castellan
DB_PORT ?= 5432
DB_HOST ?= localhost
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_DATABASE)?sslmode=disable
PG_DSN ?= postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)

migrate:
	PATH="$(HOME)/go/bin:$(PATH)" GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" GOOSE_MIGRATION_DIR=migrations goose up -s

migrate-down:
	PATH="$(HOME)/go/bin:$(PATH)" GOOSE_DRIVER=postgres GOOSE_DBSTRING="$(DATABASE_URL)" GOOSE_MIGRATION_DIR=migrations goose down -s

# Drop and recreate the database
db-drop:
	@psql "$(PG_DSN)/postgres" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$(DB_DATABASE)' AND pid <> pg_backend_pid();" -c "DROP DATABASE IF EXISTS $(DB_DATABASE);" -c "CREATE DATABASE $(DB_DATABASE);"

# Full reset: drop, migrate, seed
db-reset: db-drop migrate seed

# Test the application (mirrors unit-testing.yml: race + coverage)
test:
	@echo "Testing..."
	@go test -race -count=1 ./...
# Integration Tests (mirrors integration-testing.yml)
itest:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./internal/provider/... ./internal/database/... ./internal/gateway/... ./internal/ledger/... ./internal/settlement/...

# Security checks (mirrors security.yml: govulncheck + gosec)
security:
	@echo "Running security checks..."
	@govulncheck ./...
	@gosec -confidence medium ./...

# Trivy vulnerability scan (mirrors trivy.yml; requires Docker + trivy CLI)
trivy-scan: build-linux
	@echo "Running Trivy scan..."
	@docker build -t castellan:ci .
	@trivy image --severity HIGH,CRITICAL --exit-code 1 --ignore-unfixed castellan:ci

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main bin/castellan-api bin/castellan-worker

# Live Reload
watch:
	@powershell -ExecutionPolicy Bypass -Command "if (Get-Command air -ErrorAction SilentlyContinue) { \
		air; \
		Write-Output 'Watching...'; \
	} else { \
		Write-Output 'Installing air...'; \
		go install github.com/air-verse/air@latest; \
		air; \
		Write-Output 'Watching...'; \
	}"

.PHONY: all ci ci-full build lint vet test itest security trivy-scan run clean watch docker-run docker-down build-linux build-worker run-worker docs validate-docs migrate migrate-down db-drop db-reset
