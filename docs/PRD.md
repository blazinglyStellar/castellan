# Castellan — Product Requirements Document

## Vision
Enable software systems, APIs, and AI agents to transact economically with minimal friction through programmable, usage-based payment infrastructure. Long-term: programmable economic middleware for software systems and AI agents.

## Problem
Most API providers rely on monthly subscriptions, Stripe billing, manual invoicing, or unsustainable free tiers. These models create pain on all sides:

**For API providers:** Difficult billing management, Stripe overhead, abuse from free users, poor monetization for low-volume APIs, inability to monetize small utilities, payment restrictions in emerging markets.

**For API consumers:** Overpaying for unused plans, onboarding friction, unnecessary subscriptions, inability to pay proportionally to usage, fragmented billing systems.

**For AI systems:** APIs are not machine-payable, autonomous payment flows are difficult, tool usage billing lacks standardization, no native infrastructure for economic agents.

## Target Users
- Independent API developers (weather, translation, image processing, scraping, analytics APIs)
- OSS maintainers exposing APIs and services
- AI infrastructure builders building agents, tool ecosystems, and autonomous workflows

## Core Product Concept
Castellan acts as a programmable gateway layer positioned in front of APIs. It authenticates consumers, meters usage, enforces pricing, verifies balances, handles settlement accounting, and serves analytics — all as a drop-in reverse proxy requiring no API rewrites.

```
Client → Castellan Gateway → Provider API
```

## High-Level Architecture

### MVP Architecture Diagram
```
Client
  ↓
Castellan Gateway
  ├── Auth Layer
  ├── Metering Engine
  ├── Pricing Engine
  ├── Ledger Service
  ├── Usage Logger
  └── Proxy Engine
  ↓
Provider API

Background Workers
  ├── Settlement Worker
  ├── Usage Aggregator
  └── Reconciliation Worker

Databases
  ├── PostgreSQL
  └── Redis

Blockchain Layer
  └── Stellar Network
```

### Core Components

**1. Gateway Service** — Core Go reverse proxy (net/http + Chi, `httputil.ReverseProxy`). Request lifecycle: receive → authenticate → resolve endpoint → determine pricing → validate balance → reserve funds → forward → capture response → commit usage → return response. Features: header injection (consumer IDs, usage metadata, tracing), configurable upstream timeouts, basic retry for transient failures, request size limits, JSON structured logging.

**2. Authentication Layer** — API key auth via `Authorization: Bearer ca_xxx`. Validates existence, status, expiration. Resolves associated user, account, balance, permissions. Keys are SHA-256 or bcrypt hashed — never stored raw. Supports rotation and instant revocation. Future: wallet signatures, capability tokens with scoped permissions (max daily spend, allowed APIs, expiry).

**3. Pricing Engine** — Fixed per-request pricing per endpoint (`price_amount`). Resolution: match route → resolve provider → fetch pricing config → return billable amount. Example: `/search → 0.0001 XLM`, `/weather → 0.0002 XLM`. Future: per-compute-unit, per-token, per-MB, dynamic load pricing, subscription hybrid.

**4. Metering Engine** — Deterministic, idempotent, auditable request accounting. Tracks: request count, latency, response size, billing amount, status codes. Each billable request generates a `usage_event` with a unique `request_id` for idempotency. Metering flow: request accepted → cost reserved → upstream response received → usage event persisted → reservation finalized.

**5. Ledger Service** — Internal accounting tracking balances, deductions, reservations, settlements. Prepaid balances only — deduct before forwarding. Balance management with temporary reservations during request execution. Failed requests release reservations (refunds). Entry types: deposit, reservation, deduction, refund, settlement. Ledger flow: balance=10, reserve=0.1, available=9.9 → request succeeds: deduct → request fails: release.

**6. Rate Limiting** — Redis token bucket. Keys: `rate:{consumer_id}:{endpoint_id}`. Configurable per endpoint (e.g. 100 requests/minute). Consumer-level and endpoint-level limits.

**7. Settlement Service** — Batched (not per-request blockchain). Aggregates provider earnings, creates settlement batches, executes Stellar transfers, reconciles results. Scheduling: every 5 minutes or threshold-triggered. Stellar features: wallet generation, validation, transaction submission/monitoring, memo support. Assets: XLM initially, USDC on Stellar later. Stellar Go SDK.

**8. Deposit Infrastructure** — Single Castellan-operated hot wallet receiving deposits. Memo-based routing with unique UUID per consumer (same pattern as exchanges). Flow: consumer requests deposit → SEP-7 URI generated (`web+stellar:pay?destination=...&memo=uuid&memo_type=text`) → dashboard renders QR code (`qrcode.react`) + raw address/memo → consumer scans (wallet pre-fills) or copy-pastes → sends XLM → deposit watcher polls Stellar every ~5s → payment matched by memo, UNIQUE `tx_hash` prevents replay → internal ledger credited. 5 XLM minimum threshold. Memo recovery via manual support.

**9. Dashboard** — Next.js 15, Tailwind, shadcn/ui, React Query. **Provider:** API registration (CLI bulk + OpenAPI sync deferred to Phase 3), per-endpoint pricing configuration, usage analytics, earnings overview, settlement history. **Consumer:** wallet balance, deposit history, usage history, API consumption logs.

## Stack

| Layer | Choice |
|-------|--------|
| Backend | Go (net/http, Chi, pgx, sqlc, go-redis, zap, viper, cobra) |
| DB | PostgreSQL (ledger, events, config), Redis (rate limits, cache, reservations) |
| Frontend | Next.js 15, Tailwind, shadcn/ui, React Query |
| Observability | Prometheus, Grafana, OpenTelemetry |
| Blockchain | Stellar Go SDK, XLM (USDC later) |
| Deployment | Docker Compose (Kubernetes later) |

## Data Model (10 tables)

**users** — id (UUID PK), email (unique), deposit_memo (unique), payout_stellar_address, created_at, updated_at
**api_keys** — id, user_id (FK→users), key_hash, label, status (active/revoked/expired), created_at, expires_at
**providers** — id, owner_id (FK→users), name, base_url, status (active/inactive/suspended), created_at, updated_at
**api_endpoints** — id, provider_id (FK→providers), route, method, price_amount (NUMERIC), currency (XLM/USDC), rate_limit, status (active/inactive/draft), created_at, updated_at. Unique: (provider_id, route, method)
**accounts** — id, owner_id (FK→users, unique), balance (NUMERIC), currency, created_at, updated_at. 1:1 with users.
**ledger_entries** — id, account_id (FK→accounts), entry_type (deposit/reservation/deduction/refund/settlement), amount, balance_after, currency, reference_id, reference_type, status, description, created_at
**usage_events** — id, consumer_id (FK→users), provider_id (FK→providers), endpoint_id (FK→api_endpoints), request_cost, currency, status_code, latency_ms, response_size, request_id (unique), status, created_at
**deposits** — id, account_id (FK→accounts), from_address, amount, currency, memo, tx_hash (unique), status, created_at, confirmed_at
**settlement_batches** — id, status (pending/processing/completed/failed), total_amount, currency, entry_count, tx_hash, created_at, completed_at
**settlement_entries** — id, batch_id (FK→settlement_batches), provider_id (FK→providers), amount, currency, wallet_address, status, created_at

All FK relationships use ON DELETE CASCADE (acceptable for MVP — production should use soft-delete). Stellar wallet references inlined on users (deposit_memo, payout_stellar_address) avoiding a separate table.

### Table Groupings
- **Authentication:** users → api_keys
- **Gateway Config:** users → providers → api_endpoints
- **Finance (Internal):** users → accounts → ledger_entries, deposits
- **Finance (On-chain):** users.deposit_memo + users.payout_stellar_address (inlined)
- **Metering:** users + providers + api_endpoints → usage_events
- **Settlement:** settlement_batches → settlement_entries → providers

## Key Design Decisions

- **No blockchain per request** — usage aggregated in internal ledger, batched settlement keeps latency low and fees minimal
- **Prepaid balances** — deduct before forwarding to prevent negative balances and enable sub-100ms auth decisions
- **Single hot wallet** for deposits — memo-based routing (like exchanges), avoids managing per-user wallets
- **Internal credit != on-chain balance** — `accounts.balance` is DB-owned, fast to read. Gateway never queries Stellar during request processing. Stellar owns the source of truth for on-chain balances
- **`balance_after` in ledger_entries** — point-in-time balance query without summing full history; also acts as consistency check
- **`request_id` UNIQUE** — idempotency key prevents double-billing on retry after crash
- **Polymorphic reference** (`reference_id` + `reference_type`) — one ledger entry can reference usage_event, deposit, or settlement_batch without three nullable FK columns
- **`provider_id` denormalized on usage_events** — eliminates frequent join for "total provider earnings" queries

## Authentication & Authorization

**Initial (MVP):** API keys with SHA-256/bcrypt hashing. Session tokens for temporary scoped access.

**Future:** Wallet signatures for cryptographic request authorization. Capability tokens with scoped permissions:
```json
{ "max_daily_spend": 5, "allowed_apis": ["weather", "search"], "expires_at": 123456789 }
```

## Payment Model
Prepaid balances: user deposits funds → balance credited internally → requests deduct credits → providers settled periodically. Advantages: low latency, simpler architecture, reduced chain interactions.

## Phases

### Phase 1 — Core Gateway (MVP)
**Goal:** Validate core usage-based billing concept.
**Deliverables:** Go reverse proxy gateway, request metering, fixed pricing rules, in-memory ledger, API key auth, usage logging, minimal dashboard.
**Stack:** Go, PostgreSQL, Redis, Docker.
**Success:** Provider can wrap an API, requests metered, balances deduct correctly, dashboard reflects usage.

### Phase 2 — Real Payment Infrastructure (MVP)
**Goal:** Integrate real settlement.
**Deliverables:** Stellar wallet integration, prepaid balances, batched settlement, provider payouts, transaction history, deposit watcher, SEP-7 QR deposits.
**Additional tech:** Stellar SDK, queue workers.
**Success:** Users deposit funds, providers receive payouts, settlements reconcile correctly.

### Phase 3 — Developer Platform
**Goal:** Improve adoption and usability.
**Deliverables:** JS/Go/Python SDKs, provider onboarding flows, dashboard analytics, pricing management, CLI tooling (`castellan import ./openapi.yaml`, `castellan login`, `castellan providers`), OpenAPI 3.0/3.1 spec sync.
**Success:** Third-party developers integrate independently, APIs onboard with minimal support.

### Phase 4 — Marketplace & Ecosystem
**Goal:** Create API economy infrastructure.
**Deliverables:** API discovery, searchable marketplace, usage reputation, provider profiles, OSS funding integrations.
**Success:** Multiple APIs using platform, discoverability ecosystem exists.

### Phase 5 — AI-Agent Infrastructure
**Goal:** Enable autonomous software commerce.
**Deliverables:** Agent wallets with delegated spending (human-controlled parent wallets authorize agents), spending policies (max daily spend, approved tools, budget caps), capability tokens, programmable policies, session-based payment streams, machine-readable pricing schemas.
**Success:** AI agents autonomously consume APIs with spending constraints enforced.

## API Examples

```
POST /providers                          — Register provider
POST /providers/:id/endpoints            — Register endpoint
POST /wallet/deposit                     — Deposit funds
GET  /proxy/{provider}/{route}           — Proxy request (Auth: Bearer ca_xxx)
POST /api/v1/providers/{id}/endpoints/bulk — Bulk-import from CLI
```

## MVP Success Criteria

**Phase 1:** Providers register APIs → requests proxy correctly → pricing resolves → balances deduct accurately → usage events persist reliably. Stable request handling, acceptable latency, structured logging functional.

**Phase 2:** Deposits credit correctly → settlement batches execute → providers receive payouts → reconciliation succeeds. Stellar integration stable, failed settlements recover safely, ledger consistency maintained.

## MVP Out of Scope
Dynamic pricing, multi-chain payments, autonomous AI wallets, decentralized execution/governance, cryptographic capability tokens, token streaming/streaming payments, token economies, advanced analytics, SLA guarantees, enterprise compliance, consumer paywalls, creator tipping, fiat on-ramping, NFT ecosystems, complex DeFi integrations.

## Expected Success Metrics

**Technical:** Gateway latency (p95), settlement accuracy, throughput (requests/sec), uptime.
**Product:** APIs onboarded, monthly requests, total payment volume, active developers, SDK adoption.

## Security Requirements
- HTTPS only
- API key hashing (SHA-256 or bcrypt, never stored raw)
- Replay protection
- Input validation + SQL injection protection
- Request size limits
- Rate limiting (consumer-level + endpoint-level)
- Atomic ledger operations with transactional deduction flow
- Immutable ledger entries
- Settlement audit logs

## Recommended Repository Structure
```
castellan/
├── cmd/
│   ├── gateway/
│   ├── worker/
│   └── cli/              # Phase 3
├── internal/
│   ├── auth/
│   ├── billing/
│   ├── gateway/
│   ├── ledger/
│   ├── metering/
│   ├── pricing/
│   ├── provider/
│   ├── settlement/
│   ├── storage/
│   ├── wallet/
│   └── worker/
├── migrations/
├── deployments/
├── dashboard/            # Next.js (same repo)
├── docs/
├── sdk/                  # Phase 3
│   ├── js/
│   ├── go/
│   └── python/
└── examples/
```

## Competitive Landscape
**API Gateways:** Kong, Tyk, AWS API Gateway. **Payment Infrastructure:** Stripe Billing, Coinbase Commerce, Superfluid. **Blockchain:** Lightning-based API monetization, L402 ecosystems.

## Differentiators
- Developer-first UX: `castellan wrap https://myapi.com`, drop-in proxying, no API rewrites. Minimal integration: one CLI command to wrap an API
- Usage-based granular billing vs rigid subscription tiers
- Low-cost programmable payouts via Stellar (transaction fees fractions of a cent)
- Architected for autonomous AI-agent systems from day one (delegated wallets, spending policies, machine-readable pricing)
- OSS-aligned: Apache 2.0 or MIT, potential modular repos (gateway core, SDKs, settlement service, dashboard, CLI)

## Risks and Mitigations
- **Latency overhead** — Gateway must remain performant; use lightweight Go, avoid per-request blockchain calls
- **Fraud/abuse** — Replay protection, rate limiting, request signing (future)
- **Pricing complexity** — Start with fixed per-request pricing, expand later
- **Blockchain dependency** — Resilient fallback handling, internal ledger decouples gateway from Stellar availability
- **Developer adoption** — DX quality is critical; invest in CLI, SDKs, and onboarding
- **Market education** — Clear positioning as usage-based infrastructure, not payment processor
- **Premature AI focus** — Avoid over-optimizing for speculative workflows in MVP

## Final Technical Priorities (MVP)
1. Ledger correctness
2. Accurate metering
3. Low operational complexity
4. Reliable settlement
5. Excellent developer experience
6. Extensible architecture

The MVP's primary purpose is validating whether developers want programmable, usage-based API monetization infrastructure. Simplicity, reliability, and excellent DX take precedence over advanced features.
