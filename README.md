# Castellan

**Usage-based API monetization gateway** — a programmable reverse proxy that enables developers to charge for API access per request instead of subscriptions, with Stellar-based settlement and a full web dashboard.

---

## Why Castellan?

**The problem.** Most API monetization relies on monthly subscriptions, Stripe billing, or unsustainable free tiers. This model is rigid: consumers overpay for unused capacity, small APIs can't justify a billing system, and there is no infrastructure for machine-to-machine or AI-agent payments. API providers face overhead just to meter usage, manage subscriptions, and reconcile payments.

**The solution.** Castellan sits in front of existing APIs as a lightweight Go reverse proxy. It authenticates consumers via API keys, resolves per-endpoint pricing, validates prepaid balances, forwards requests to upstream providers, and records every billable event in an auditable PostgreSQL ledger. Rather than executing blockchain transactions per request, usage is aggregated internally and settled to provider Stellar wallets in periodic batches — keeping gateway latency low while enabling transparent, low-cost financial settlement.

**The vision.** Castellan evolves into programmable economic middleware for software systems — enabling APIs, scripts, and AI agents to transact autonomously with minimal friction.

---

## Request Lifecycle

A successful proxied request flows through these stages:

```
Client → Auth Middleware → Pricing Resolution → Balance Validation → Rate Limiter
    → Upstream Proxy (with timeout/retry) → Usage Event Commit → Ledger Deduction → Client
```

1. Gateway receives the request and extracts the API key from `Authorization: Bearer ca_xxx`
2. Auth middleware validates the key (hash lookup, status check, expiration) and resolves the consumer identity
3. Pricing engine matches the request route + method to the provider's endpoint configuration to determine cost
4. Ledger service checks the consumer's prepaid balance and atomically reserves funds
5. Rate limiter checks Redis token buckets per consumer and per endpoint
6. Reverse proxy forwards the request to the upstream provider API with injected tracing headers
7. Response is captured; a usage event is persisted (consumer, provider, endpoint, cost, status code, latency, request ID)
8. Ledger commits the reservation as a deduction (or releases it on upstream failure)
9. Response is returned to the client

All stages produce structured JSON log lines with correlation IDs for observability.

---

## Architecture

```
┌──────────────┐     ┌─────────────────────────────────────────────────────┐
│   Consumer   │────▶│                  Castellan Gateway                    │
│  (API Key)   │     │  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐  │
└──────────────┘     │  │  Auth   │ │ Pricing  │ │ Ledger   │ │ Proxy  │  │
                     │  │ Layer   │ │ Engine   │ │ Service  │ │ Engine │  │
                     │  └─────────┘ └──────────┘ └──────────┘ └────────┘  │
                     └────────────────────┬────────────────────────────────┘
                                          │
                                          ▼
                              ┌──────────────────────┐
                              │   Provider API        │
                              │   (upstream service)  │
                              └──────────────────────┘

┌────────────────────────────────────────────────────────────────────────────┐
│                         Background Workers                                  │
│  ┌─────────────────┐  ┌──────────────────┐  ┌─────────────────────────┐   │
│  │ Settlement       │  │ Deposit Watcher   │  │ Usage Aggregator       │   │
│  │ (batched Stellar)│  │ (memo-based XLM)  │  │ (earnings computation) │   │
│  └─────────────────┘  └──────────────────┘  └─────────────────────────┘   │
└────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────┐
│                              Data Layer                                     │
│  ┌────────────────┐  ┌──────────────┐  ┌───────────────────────────────┐  │
│  │   PostgreSQL    │  │    Redis      │  │   Stellar Network            │  │
│  │  (users, keys,  │  │  (rate limit  │  │  (XLM deposits, payouts)     │  │
│  │   providers,    │  │   token       │  │                              │  │
│  │   endpoints,    │  │   buckets,    │  │                              │  │
│  │   ledger,       │  │   reservations)│  │                              │  │
│  │   settlements)  │  │               │  │                              │  │
│  └────────────────┘  └──────────────┘  └───────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────┘
```

### Core Components

#### Gateway Service (Go)
HTTP server using Chi router with middleware chain: auth → pricing → ledger → rate-limit → proxy → usage-commit. Uses `httputil.ReverseProxy` for efficient upstream forwarding with configurable timeouts, retry policies, and request size limits. Structured logging via zap with request ID correlation.

#### Pricing Engine
Endpoint-based fixed per-request pricing. Each provider registers routes with a `price_amount` and `currency` (XLM or USDC). Pricing is resolved by matching `(provider_id, route, method)` and cached for low-latency lookups.

#### Ledger Service
Internal accounting system supporting five entry types: `deposit`, `reservation`, `deduction`, `refund`, `settlement`. Implements an atomic reserve-commit-refund flow: funds are temporarily reserved during upstream request execution, committed on success, or released on failure. All entries record `balance_after` for full auditability.

#### Metering Engine
Generates immutable usage events per billable request. Each event captures `(consumer_id, provider_id, endpoint_id, request_cost, status_code, latency_ms, response_size, request_id)`. Idempotent — duplicate `request_id` values are rejected.

#### Rate Limiter
Redis-backed token bucket algorithm. Configurable per-consumer and per-endpoint limits (e.g., 100 requests/minute). Bucket state stored in Redis keys: `rate:{consumer_id}:{endpoint_id}`.

#### Deposit Watcher
Background worker polling Stellar Horizon for incoming payments to a Castellan-operated hot wallet. Payments are matched by memo UUID and credited to the consumer's internal balance. Uses `UNIQUE tx_hash` constraint for replay protection. Generates SEP-7 URIs for wallet-compatible QR codes.

#### Settlement Worker
Cobra-command background worker running on a configurable interval (default ~5 min). Each cycle: aggregates unsettled per-provider earnings from the ledger, creates a settlement batch with entries, submits Stellar payout transactions, and marks earnings as settled.

---

## Data Model

### Core Tables

| Table | Purpose | Key Columns |
|---|---|---|
| `users` | Consumer and provider identities | `email`, `deposit_memo`, `payout_stellar_address` |
| `api_keys` | Hashed authentication credentials | `user_id`, `key_hash`, `status`, `expires_at` |
| `providers` | Upstream API registrations | `owner_id`, `name`, `base_url`, `status` |
| `api_endpoints` | Per-route pricing configuration | `provider_id`, `route`, `method`, `price_amount`, `currency` |
| `accounts` | Prepaid balance wallets | `owner_id`, `balance`, `currency` |
| `ledger_entries` | Immutable accounting trail | `account_id`, `entry_type`, `amount`, `balance_after`, `reference_id` |
| `usage_events` | Billable request records | `consumer_id`, `provider_id`, `endpoint_id`, `request_cost`, `request_id` |
| `deposits` | Stellar payment tracking | `account_id`, `from_address`, `amount`, `memo`, `tx_hash`, `status` |
| `settlement_batches` | Aggregated payout groups | `total_amount`, `entry_count`, `tx_hash`, `status` |
| `settlement_entries` | Per-provider payout records | `batch_id`, `provider_id`, `amount`, `wallet_address` |

### Ledger Entry Types

| Type | Direction | Description |
|---|---|---|
| `deposit` | + | Consumer funds their account via Stellar |
| `reservation` | - | Temporary hold during request execution |
| `deduction` | - | Final charge after successful upstream response |
| `refund` | + | Release of reservation on upstream failure |
| `settlement` | - | Transfer of provider earnings to Stellar wallet |

---

## Features (Backlog)

| # | Epic | Description |
|---|---|---|
| 1 | **Gateway Request Lifecycle** | Authenticate, price, validate balance, rate-limit, proxy, and commit usage events with sub-100ms overhead. Includes Chi router scaffolding, structured JSON logging, request context propagation, reverse proxy with response capture, idempotent usage event generation, configurable timeouts/retries/size limits, and end-to-end integration tests. |
| 2 | **API Key Authentication** | PostgreSQL `api_keys` table with sqlc queries, bcrypt/SHA-256 key hashing and generation, HTTP endpoints for CRUD management, gateway auth middleware extracting keys from `Authorization: Bearer` headers, and session token support for temporary scoped access. |
| 3 | **Provider & Endpoint Management** | PostgreSQL migrations for providers and api_endpoints, sqlc-generated CRUD queries, HTTP handlers for registering providers and configuring per-endpoint pricing (route, method, price_amount, currency, rate_limit). |
| 4 | **Prepaid Balance Ledger** | Atomic reserve-commit-refund ledger service with PostgreSQL transactions. Every entry records `balance_after` for audit trail. HTTP handlers for balance queries and entry history. |
| 5 | **Rate Limiting** | Redis-backed token bucket algorithm. Configurable per-consumer and per-endpoint limits. Chi middleware enforcing limits with Viper-based configuration. |
| 6 | **Stellar Deposits** | Deposit intent API returning SEP-7 URIs. QR code generation via `qrcode.react`. Background worker polling Stellar Horizon every ~5s, matching payments by memo UUID, crediting internal balances atomically with replay protection via `UNIQUE tx_hash`. Minimum deposit threshold (5 XLM) to prevent dust. |
| 7 | **Batched Settlement** | Migration + sqlc for settlement schema. Cobra command running a worker on ~5 min intervals. Each cycle aggregates unsettled per-provider earnings, submits Stellar payouts, records entries, and marks ledger earnings settled to prevent double payment. |
| 8 | **Web Dashboard** | Next.js 15 dashboard with shadcn/ui, React Query, and Tailwind. Provider views: earnings overview, usage analytics, settlement history, API registration forms, endpoint pricing configuration. Consumer views: wallet balance, deposit history (with QR code display), usage logs. Read-only API endpoints on the Go gateway back the dashboard data layer. |

---

## Stack

| Layer | Technology |
|---|---|
| **Gateway** | Go 1.26, Chi router, httputil.ReverseProxy |
| **Database** | PostgreSQL, pgx v5 via database/sql, sqlc (codegen), goose (migrations) |
| **Cache** | Redis (rate limiting, temporary reservations) |
| **Settlement** | Stellar network (XLM), SEP-7 URIs |
| **Frontend** | Next.js 15 (App Router), Tailwind CSS, shadcn/ui, React Query |
| **Logging** | zap structured JSON logging |
| **Infrastructure** | Docker Compose (Postgres, Redis, Gateway) |

---

## Getting Started

```bash
# Start infrastructure (Postgres + Redis)
make docker-run

# Run database migrations
DATABASE_URL="postgres://castellan:castellan@localhost:5432/castellan?sslmode=disable" \
  goose -s -dir migrations postgres "$DATABASE_URL" up

# Start the gateway server (default :8080)
make run

# Run unit tests
make test

# Run integration tests (requires Docker)
make itest

# Run full suite with race detection
go test -race -count=1 ./...

# Frontend
cd dashboard && npm run dev
```

See `Makefile` for all commands.

---

## Project Structure

```
castellan/
├── cmd/
│   └── gateway/          # Go server entrypoint
├── internal/
│   ├── database/         # pgx connection pool, health checks
│   ├── server/           # HTTP server, routes, middleware
│   └── repository/       # sqlc-generated query layer (db/ + query/)
├── migrations/           # Goose SQL migrations
├── dashboard/            # Next.js 15 web dashboard
├── docs/                 # PRD, MVP spec, architecture docs
├── docker-compose.yml    # Postgres + Redis for local dev
└── sqlc.yaml             # sqlc codegen config
```
