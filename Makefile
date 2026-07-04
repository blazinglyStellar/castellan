# Simple Makefile for a Go project

# Default: run all CI checks, then build
all: ci build

build:
	@echo "Building..."
	@go build -o main.exe cmd/api/main.go

# Aggregate CI checks (runs before build via `all`)
ci: lint vet test security
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
# Create DB container
docker-run:
	@docker compose up --build

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

# Test the application (mirrors unit-testing.yml: race + coverage)
test:
	@echo "Testing..."
	@go test -race -count=1 -covermode=atomic -coverprofile=coverage.out ./... -v
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
trivy-scan:
	@echo "Running Trivy scan..."
	@docker build -t castellan:ci .
	@trivy image --severity HIGH,CRITICAL --exit-code 1 --ignore-unfixed castellan:ci

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

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

.PHONY: all ci ci-full build lint vet test itest security trivy-scan run clean watch docker-run docker-down build-worker run-worker
