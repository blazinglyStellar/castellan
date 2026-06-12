# FlowGate

**Usage-based API monetization gateway** — metering, prepaid billing, and Stellar-powered settlement for developers.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql)](https://postgresql.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue)](LICENSE)
![Status](https://img.shields.io/badge/Status-MVP-yellow)
[![codecov](https://codecov.io/gh/mustapha-aminu-abubakar/flowgate/branch/main/graph/badge.svg)](https://codecov.io/gh/mustapha-aminu-abubakar/flowgate)

---

## Why Castellan?

Today, every API provider that wants to charge per-request either builds a custom billing system, forces users into rigid subscriptions, or just gives it away for free. Subscription billing means overpaying or underutilizing. Stripe overhead kills small APIs. And there's no infrastructure for machine-to-machine payments — APIs can't pay each other, and AI agents have no way to transact economically.

Castellan standardizes that layer:

- **Per-request pricing** — each endpoint has a fixed cost. No subscriptions, no tiers, no surprises.
- **Prepaid balances** — consumers fund an internal wallet. The gateway checks balance on every request, deducts in real time.
- **Batched Stellar settlement** — usage aggregates in an internal ledger. Provider payouts are batched and settled on Stellar. No per-request blockchain transactions.
- **Zero billing infrastructure** — providers don't build invoicing, reconciliation, or payout systems. Castellan handles it.

---

## What We're Building

### 1. Go Reverse Proxy Gateway

The core request lifecycle: authenticate, price, validate balance, forward, meter, deduct. All in a single Go binary using `httputil.ReverseProxy`.

**Key deliverables:**
- API key authentication with hashed bearer tokens
- Route-based pricing resolution
- Prepaid balance validation per request
- Request forwarding with header injection and timeouts
- Structured JSON logging with correlation IDs
- Usage event persistence (idempotent, auditable)

### 2. Prepaid Ledger Engine

Internal accounting system that tracks balances, reservations, deductions, and refunds — without touching the blockchain on every request.

**Key deliverables:**
- Balance management with fast local reads
- Temporary fund reservations during request execution
- Atomic deduction flow (reserve → forward → commit)
- Automatic refunds on upstream failures
- Immutable ledger entry audit trail

### 3. Stellar Settlement Infrastructure

Batched provider payouts over the Stellar network. A deposit watcher monitors incoming payments and credits internal balances. A settlement worker aggregates earnings and executes batched transfers.

**Key deliverables:**
- SEP-7 QR code deposit flow (destination + memo auto-filled in wallet)
- Deposit watcher polling Stellar Horizon every ~5s
- Settlement batch creation and Stellar transaction execution
- Provider payout reconciliation
- Memo-based payment routing

### 4. Background Workers

Independent Go services that handle async operations without blocking the gateway.

**Key deliverables:**
- Settlement worker — aggregate earnings, create batches, submit Stellar txs
- Deposit watcher — monitor incoming payments, credit balances
- Usage aggregator — prepare provider settlement data

### 5. Next.js Dashboard

A full-featured dashboard for both providers and consumers, built with Next.js 15 and shadcn/ui.

**Key deliverables:**
- Provider dashboard — API registration, pricing configuration, usage analytics, earnings overview, settlement history
- Consumer dashboard — wallet balance, deposit flow with SEP-7 QR code, usage history, API consumption logs
- Role-aware sidebar adapting nav items for Provider, Consumer, or Both

---

## MVP Scope

| In scope | Deferred |
|---|---|
| Fixed per-request pricing | Dynamic / usage-based pricing |
| API key authentication | OAuth, JWTs, capability tokens |
| Internal prepaid ledger | On-chain balance reads |
| Batched Stellar settlement | USDC, multi-chain |
| SEP-7 QR deposits | Streaming payments |
| Redis rate limiting | Advanced analytics |
| Provider + Consumer dashboards | AI-agent wallets, marketplaces |

---

## Architecture

```
Client
  ↓
Castellan Gateway
  ├── Auth Layer          → API key validation
  ├── Metering Engine     → Per-request accounting
  ├── Pricing Engine      → Route-based cost resolution
  ├── Ledger Service      → Balance mgmt, reservations, deductions
  └── Proxy Engine        → httputil.ReverseProxy forwarding
  ↓
Provider API

Background Workers
  ├── Settlement Worker   → Aggregate earnings, execute Stellar payouts
  ├── Deposit Watcher     → Monitor Stellar for incoming payments
  └── Usage Aggregator    → Prepare provider settlement data
```

Blockchain transactions happen in **batches**, not per request. Usage aggregates in the internal ledger; Stellar settlement runs asynchronously.

---

## Quick Start

```bash
# Prerequisites: Go 1.26+, Docker & Docker Compose

# Clone and start infrastructure
git clone https://github.com/mustapha-aminu-abubakar/castellan.git
cd castellan
docker compose up -d postgres redis

# Run migrations
goose -s -dir migrations postgres "postgres://postgres:postgres@localhost:5432/castellan?sslmode=disable" up

# Start gateway
go run ./cmd/api
```

**Dashboard:**
```bash
cd dashboard
npm install
npm run dev    # → http://localhost:3000
```

---

## Commands

| Command | What it does |
|---|---|
| `make build` | Build to `main.exe` |
| `make test` | Full test suite (`go test -race -count=1 ./...`) |
| `make watch` | Live reload via `air` |
| `make run` | Run the API server |

See [`AGENTS.md`](AGENTS.md) for the full command reference.

---

## Project Structure

```
castellan/
├── cmd/api/               # Gateway HTTP server entrypoint
├── internal/
│   ├── repository/        # sqlc queries + generated Go
│   └── server/            # HTTP server, routes, handlers
├── migrations/            # goose migration files
├── dashboard/             # Next.js 15 frontend
├── docs/                  # PRDs, schema docs, design analysis
├── docker-compose.yml     # Postgres + Redis + app
└── Dockerfile             # Go build image
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| Gateway | Go (`net/http`, `httputil.ReverseProxy`) |
| Database | PostgreSQL 16 + Redis 7 |
| Settlement | Stellar Network |
| Dashboard | Next.js 15, Tailwind CSS, shadcn/ui |
| Query Layer | sqlc (type-safe Go from SQL) |
| Migrations | goose |

---

## Philosophy

**Usage billing should be infrastructure, not a feature.** Castellan treats API monetization the way Stripe treats payments — as a standardized, pluggable layer that developers shouldn't have to build themselves.

- **Transactions off the critical path** — the blockchain is the settlement layer, not the request path. Speed and cost stay predictable.
- **Honest scope** — this doesn't solve identity, reputation, or discovery. It solves metering, billing, and settlement. Each provider defines their trust model.
- **Machine-payable by design** — the architecture is built for APIs and eventually AI agents to transact economically without human intervention.
- **Pragmatic layering** — use the database for speed (balance checks, usage queries) and the chain for what it's good at (immutable settlement, transparent payouts).

---

## Contributing

PRs welcome. This is an early-stage MVP — code churn is expected.

```bash
make test
```

See [`CONTRIBUTING.md`](CONTRIBUTING.md) and [`docs/`](docs/) for design context before making schema changes.

---

## License

Apache 2.0
