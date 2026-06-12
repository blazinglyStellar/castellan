# Castellan MVP PRD (Short)

Usage-based API monetization gateway. Go reverse proxy that authenticates, meters, prices, and bills API requests with Stellar settlement.

## MVP Scope

- **Phase 1** — Core gateway: auth, routing, metering, pricing, ledger
- **Phase 2** — Payments: deposits, batched Stellar settlement

## Architecture

```
Client → Gateway (Auth → Pricing → Ledger → Proxy) → Provider API
         Background: Settlement Worker, Deposit Watcher
         DB: PostgreSQL + Redis → Stellar Network
```

## Key Components

| Component | Role |
|-----------|------|
| Gateway | Reverse proxy: auth → balance check → forward → log usage |
| Auth | API keys (`Authorization: Bearer ca_xxx`), SHA-256 hashed |
| Pricing | Fixed per-request cost per endpoint (e.g. `/search → 0.0001 XLM`) |
| Ledger | Internal accounting: deposits, reservations, deductions, refunds, settlements |
| Metering | Tracks request count, latency, cost, status — idempotent & auditable |
| Settlement | Batched every ~5 min: aggregate earnings → Stellar tx → provider wallet |
| Deposits | SEP-7 URI + QR code, single hot wallet with memo routing, 5 XLM minimum |
| Rate Limiting | Redis token bucket, per consumer + endpoint |

## Stack

- **Go**: chi, httputil.ReverseProxy, pgx, sqlc, go-redis, zap, viper, cobra
- **DB**: PostgreSQL (ledger, events, config), Redis (rate limits, cache, reservations)
- **Dashboard**: Next.js, Tailwind, React Query, shadcn/ui
- **Observability**: Prometheus, Grafana, OpenTelemetry
- **Blockchain**: Stellar Go SDK, XLM (USDC later)

## Request Lifecycle

1. Authenticate → 2. Resolve pricing → 3. Reserve funds → 4. Forward → 5. Capture response → 6. Commit usage → 7. Deduct balance

On failure: release reservation, log error, return error (no deduction).

## Key Design Decisions

- **No blockchain per request** — usage aggregated in internal ledger, batched settlement
- **Prepaid balances only** — deduct before forwarding
- **Single hot wallet** for deposits — memo-based routing (like exchanges)
- **Docker Compose** for MVP deployment

## Success Criteria

- Providers register APIs, requests proxy correctly, pricing resolves, balances deduct accurately
- Deposits credit correctly, settlement batches execute, providers receive payouts

## Out of Scope

Dynamic pricing, multi-chain, AI wallets, streaming payments, token economies, marketplaces, SLA guarantees.
