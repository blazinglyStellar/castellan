# Castellan MVP PRD
## Technical Product Requirements Document

Version: 1.0
Status: Draft
Primary Language: Golang
Scope: MVP Only

---

# 1. Overview

## Product Name
Castellan

## Summary
Castellan is a usage-based API monetization gateway that enables developers to charge for API access per request instead of relying on subscriptions or fixed-rate billing. It sits in front of existing APIs as a Go-based reverse proxy, handling authentication, request routing, metering, and billing. Each request is priced, validated against a prepaid balance, and recorded in an internal ledger, allowing precise tracking of API usage in real time.

The project solves the limitations of traditional API monetization models, which are often too rigid, inefficient, or operationally complex. Subscriptions lead to overpayment or underutilization, while free tiers are difficult to sustain. Castellan replaces these with a proportional model where users pay exactly for what they consume, making small APIs economically viable and reducing friction for both providers and consumers.

To ensure performance and scalability, Castellan does not execute blockchain transactions per request. Instead, it aggregates usage in an internal ledger and performs batched settlements using the Stellar network. This keeps the system fast while still enabling transparent, low-cost financial settlement.

Beyond billing, Castellan establishes a foundation for machine-native payments, where APIs and eventually AI agents can interact economically without human intervention. Its long-term direction is to become a programmable payment layer for software systems, enabling automated, fine-grained commerce across digital services.


## MVP Scope
This MVP focuses exclusively on:

1. Phase 1 — Core Gateway Infrastructure
2. Phase 2 — Real Payment Infrastructure

The objective is to validate:
- usage-based API monetization,
- programmable request metering,
- prepaid billing flows,
- Stellar-based settlement,
- and low-friction developer onboarding.

The MVP intentionally excludes:
- AI-agent orchestration,
- marketplaces,
- decentralized governance,
- advanced cryptographic authorization,
- streaming payments,
- token economies,
- and complex pricing models.

---

# 2. MVP Goals

## Primary Goal
Enable developers to wrap existing APIs behind a gateway that:
- meters usage,
- deducts prepaid balances,
- and settles provider earnings through Stellar.

---

## Secondary Goals

### G1 — Minimal Integration Complexity
Providers should onboard APIs with minimal infrastructure changes.

### G2 — Reliable Request Accounting
All billable requests must be tracked accurately.

### G3 — Low-Latency Gatewaying
Gateway overhead should remain operationally acceptable.

### G4 — Simple Prepaid Billing
Consumers fund balances before usage.

### G5 — Batched Settlement
Provider payouts should aggregate usage efficiently.

---

# 3. High-Level MVP Architecture

```text
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

---

# 4. System Components

# 4.1 API Gateway Service

## Description
Core Golang reverse proxy responsible for:
- request interception,
- authentication,
- balance validation,
- pricing enforcement,
- request forwarding,
- and usage recording.

---

## Responsibilities

### Request Lifecycle
1. Receive request
2. Authenticate consumer
3. Resolve provider endpoint
4. Determine endpoint pricing
5. Validate sufficient balance
6. Reserve/request billing amount
7. Forward request
8. Capture response metadata
9. Commit usage event
10. Return response

---

## Recommended Technologies

### Golang HTTP Stack
Recommended:
- net/http
- Chi router

Optional:
- Gin

---

## Proxy Layer
Recommended:
- httputil.ReverseProxy

Reason:
- native Go support,
- efficient reverse proxying,
- low-level request manipulation.

---

## Required Features

### Request Forwarding
Forward inbound requests to upstream provider APIs.

### Header Injection
Inject:
- consumer identifiers,
- usage metadata,
- tracing headers.

### Timeout Management
Configurable upstream timeouts.

### Retry Policies
Basic retry support for transient failures.

### Request Size Limits
Prevent abuse.

### Structured Logging
JSON logs for observability.

---

# 4.2 Authentication Layer

## MVP Authentication Model

### API Keys
Consumers authenticate via:

```text
Authorization: Bearer ca_xxxxx
```

---

## Responsibilities

### API Key Validation
Validate:
- existence,
- status,
- expiration.

### Consumer Resolution
Resolve associated:
- user,
- wallet,
- balance,
- permissions.

### Request Context Injection
Attach consumer metadata to request context.

---

## Database Schema

### api_keys

```sql
id UUID
user_id UUID
key_hash TEXT
status TEXT
created_at TIMESTAMP
expires_at TIMESTAMP NULL
```

---

## Security Requirements

### Store Hashed Keys Only
Never store raw keys.

Recommended:
- SHA-256
or
- bcrypt

### Rotation Support
Allow key regeneration.

### Revocation Support
Instant invalidation.

---

# 4.3 Pricing Engine

## Description
Determines request cost.

---

## MVP Pricing Model

### Fixed Per-Request Pricing
Each endpoint has:

```text
price_per_request
```

Example:

```text
/search → 0.0001 XLM
/weather → 0.0002 XLM
```

---

## Pricing Resolution Flow

1. Match route
2. Resolve provider
3. Fetch pricing config
4. Return billable amount

---

## Database Schema

### api_endpoints

```sql
id UUID
provider_id UUID
route TEXT
method TEXT
price_amount NUMERIC
currency TEXT
status TEXT
created_at TIMESTAMP
```

---

# 4.4 Metering Engine

## Description
Responsible for request accounting.

---

## Responsibilities

### Track
- request count,
- latency,
- response size,
- billing amount,
- status codes.

### Generate Usage Events
Each billable request generates:

```text
usage_event
```

---

## Metering Flow

1. Request accepted
2. Cost reserved
3. Upstream response received
4. Usage event persisted
5. Reservation finalized

---

## Important Design Requirement

Metering MUST be:
- deterministic,
- idempotent,
- auditable.

---

## Usage Event Schema

### usage_events

```sql
id UUID
consumer_id UUID
provider_id UUID
endpoint_id UUID
request_cost NUMERIC
currency TEXT
status_code INT
latency_ms INT
request_id TEXT
created_at TIMESTAMP
```

---

# 4.5 Ledger Service

## Description
Internal accounting system.

This service tracks:
- balances,
- deductions,
- reservations,
- settlements.

IMPORTANT:
The MVP uses an internal ledger.

Blockchain settlement occurs asynchronously.

---

## Core Responsibilities

### Balance Management
Track prepaid balances.

### Reservations
Temporarily reserve funds during request execution.

### Finalization
Commit deductions after successful usage recording.

### Refunds
Handle failed upstream requests.

---

## Ledger Flow

### Request Begins

```text
balance = 10
reserve = 0.1
available = 9.9
```

### Request Succeeds

```text
deduct reserved amount
```

### Request Fails

```text
release reservation
```

---

## Database Schema

### wallets

```sql
id UUID
owner_id UUID
balance NUMERIC
currency TEXT
created_at TIMESTAMP
```

---

### ledger_entries

```sql
id UUID
wallet_id UUID
entry_type TEXT
amount NUMERIC
reference_id UUID
status TEXT
created_at TIMESTAMP
```

---

## Entry Types

### Supported Types
- deposit
- reservation
- deduction
- refund
- settlement

---

# 4.6 Provider Registry

## Description
Stores upstream provider configuration.

---

## Provider Schema

### providers

```sql
id UUID
owner_id UUID
name TEXT
base_url TEXT
status TEXT
created_at TIMESTAMP
```

---

## Requirements

### Upstream Health Checking
Basic health monitoring.

### Environment Support
Support:
- local,
- staging,
- production URLs.

---

# 4.7 Usage Logging

## Requirements

### Structured Logs
JSON logging format.

### Correlation IDs
Each request receives:

```text
request_id
```

### Log Categories
- gateway
- auth
- billing
- settlement
- upstream
- worker

---

## Recommended Libraries

### Logging
- Uber Zap

### Tracing
- OpenTelemetry

---

# 5. Database Architecture

# 5.1 Primary Database

## PostgreSQL

Responsibilities:
- transactional consistency,
- ledger persistence,
- usage event storage,
- provider configuration.

---

## Recommended Features

### UUID Primary Keys
### Transactions
### Indexing
### Foreign Keys
### Row-Level Constraints

---

# 5.2 Redis

## Responsibilities

### Rate Limiting
### Temporary Reservations
### Cache Layer
### API Key Cache
### Pricing Cache

---

## Why Redis

Needed for:
- low-latency reads,
- temporary state,
- distributed throttling.

---

# 6. Request Lifecycle (Detailed)

# 6.1 Successful Request

```text
1. Client sends request
2. Gateway authenticates API key
3. Gateway resolves endpoint pricing
4. Ledger reserves funds
5. Request forwarded upstream
6. Response received
7. Usage event persisted
8. Ledger commits deduction
9. Response returned to client
```

---

# 6.2 Failed Request

```text
1. Client sends request
2. Funds reserved
3. Upstream request fails
4. Reservation released
5. Failure logged
6. Error returned
```

---

# 7. Rate Limiting

## MVP Requirements

### Consumer-Level Limits
Prevent abuse.

### Endpoint-Level Limits
Protect providers.

---

## Recommended Strategy

### Redis Token Bucket

Keys:

```text
rate:{consumer_id}:{endpoint_id}
```

---

## Initial Limits
Configurable per endpoint.

Example:

```text
100 requests/minute
```

---

# 8. Settlement Infrastructure (Phase 2)

# 8.1 Settlement Overview

Settlement transfers provider earnings from:
- internal ledger balances
- to Stellar wallets.

IMPORTANT:
Requests do NOT directly trigger blockchain transactions.

Settlement is batched.

---

# 8.2 Settlement Architecture

```text
Usage Events
  ↓
Settlement Aggregator
  ↓
Settlement Batch
  ↓
Stellar Transaction
  ↓
Provider Wallet
```

---

# 8.3 Settlement Worker

## Responsibilities

### Aggregate Provider Earnings
Compute unpaid balances.

### Create Settlement Batch
Group multiple payouts.

### Execute Stellar Transfers
Send blockchain transactions.

### Reconciliation
Verify transaction success.

---

## Worker Scheduling

Recommended:
- every 5 minutes
or
- threshold-triggered batching.

---

# 8.4 Stellar Integration

## Recommended Assets

### Initial Asset
XLM

### Future Support
USDC on Stellar.

---

## Required Features

### Wallet Generation
### Wallet Validation
### Transaction Submission
### Transaction Monitoring
### Memo Support

---

## Recommended SDK

### Stellar Go SDK

---

# 8.5 Settlement Database Schema

### settlement_batches

```sql
id UUID
status TEXT
total_amount NUMERIC
tx_hash TEXT
created_at TIMESTAMP
```

---

### settlement_entries

```sql
id UUID
batch_id UUID
provider_id UUID
amount NUMERIC
wallet_address TEXT
status TEXT
created_at TIMESTAMP
```

---

# 9. Deposit Infrastructure

# 9.1 Consumer Funding Flow

```text
1. Consumer requests deposit via dashboard
2. System returns SEP-7 URI: web+stellar:pay?destination=GABC...&memo=uuid&memo_type=text
3. Dashboard renders QR code + raw address/memo
4. Consumer scans QR (wallet pre-fills destination + memo) or copy-pastes
5. Consumer sends XLM
6. Deposit watcher polls Stellar every ~5s
7. Payment matched by memo; UNIQUE tx_hash prevents replay
8. Internal ledger credited
```

---

# 9.2 Implementation Details

## Wallet Model (MVP)

Single Castellan-operated hot wallet for receiving deposits. Memo-based routing with unique UUID per consumer — same pattern used by exchanges on Stellar.

## Minimum Deposit

5 XLM minimum threshold to prevent dust/spam attacks. Deposits below this are rejected.

## Memo Recovery

If a consumer sends XLM without a memo, the funds arrive but cannot be routed automatically. MVP handles this via manual support process: consumer provides the tx_hash, an admin verifies and credits the account.

## SEP-7 QR Code

The deposit endpoint generates a [SEP-7 URI](https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0007.md) which encodes destination address, memo type, memo value, and optionally the amount. The dashboard renders this as a QR code using `qrcode.react`. Compatible wallets (LOBSTR, Solar, Freighter, xBull) pre-fill all fields when scanned — eliminating the #1 failure mode of forgotten memos.

---

# 9.3 Deposit Watcher

## Responsibilities

### Monitor Stellar Payments
Poll Stellar Horizon: `payments?for=deposit_wallet&cursor=last_seen`

### Validate by Memo + tx_hash
Skip if: already processed (UNIQUE `deposits.tx_hash`), or amount below minimum.

### Credit Internal Balances
Insert deposit record → update `accounts.balance` → write `ledger_entry`.

---

## Recommended Design

Run as independent worker service. Poll interval: ~5 seconds (Stellar finality is ~3-5s).

# 10. Dashboard Requirements

# 10.1 Provider Dashboard

## Features

### API Registration
Providers register APIs through the dashboard UI. CLI-based bulk registration and OpenAPI spec sync are deferred to Phase 3.

### Pricing Configuration
Providers set per-endpoint pricing (`price_amount`) and toggle public/private status in the dashboard. OpenAPI extensions (`x-castellan-price`, `x-castellan-public`) for one-shot setup via CLI are deferred to Phase 3.

### Usage Analytics
### Earnings Overview
### Settlement History

---

# 10.2 Consumer Dashboard

## Features

### Wallet Balance
### Deposit History
### Usage History
### API Consumption Logs

---

# 10.3 Frontend Stack

## Recommended Technologies

### Next.js
### Tailwind CSS
### React Query
### shadcn/ui

---

# 11. Observability

# 11.1 Metrics

## Required Metrics

### Gateway Metrics
- requests/sec
- latency
- error rate

### Billing Metrics
- deductions
- settlement volume
- failed reservations

### Infrastructure Metrics
- DB latency
- Redis latency
- worker queue size

---

# 11.2 Monitoring Stack

### Prometheus
### Grafana
### OpenTelemetry

---

# 12. Security Requirements

# 12.1 Core Security

## Requirements

### HTTPS Only
### API Key Hashing
### Replay Protection
### Input Validation
### Request Size Limits
### SQL Injection Protection
### Rate Limiting

---

# 12.2 Financial Integrity

## Requirements

### Atomic Ledger Operations
### Transactional Deduction Flow
### Immutable Ledger Entries
### Settlement Audit Logs

---

# 13. Deployment Architecture

# 13.1 MVP Deployment

## Components

### gateway-service
### worker-service
### postgres
### redis
### dashboard

---

## Containerization

### Docker Compose
Initially sufficient.

---

# 13.2 Future Deployment

### Kubernetes
### Horizontal Scaling
### Queue Infrastructure

Not required for MVP.

---

# 14. Recommended Repository Structure

```text
castellan/
├── cmd/
│   ├── gateway/
│   ├── worker/
│   └── cli/           # Phase 3 — OpenAPI sync, endpoint management
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
├── dashboard/          # Next.js dashboard (same repo)
├── docs/
└── examples/
```

---

# 15. Recommended Go Libraries

## HTTP
- net/http
- chi

## Database
- pgx
- sqlc

## Caching
- go-redis

## Logging
- zap

## Config
- viper

## CLI
- cobra

## Metrics
- prometheus/client_golang

## Validation
- go-playground/validator

---

# 16. API Examples

# 16.1 Register Provider

```http
POST /providers
```

---

# 16.2 Register Endpoint

```http
POST /providers/:id/endpoints
```

---

# 16.3 Deposit Funds

```http
POST /wallet/deposit
```

---

# 16.4 Proxy Request

```http
GET /proxy/{provider}/{route}
Authorization: Bearer ca_xxx
```

---

# 17. MVP Success Criteria

# Phase 1 Success

## Functional
- providers can register APIs
- requests proxy correctly
- pricing resolves correctly
- balances deduct accurately
- usage events persist reliably

## Operational
- stable request handling
- acceptable latency
- structured logging functional

---

# Phase 2 Success

## Functional
- deposits credit correctly
- settlement batches execute
- providers receive payouts
- reconciliation succeeds

## Operational
- Stellar integration stable
- failed settlements recover safely
- ledger consistency maintained

---

# 18. Known MVP Limitations

The MVP intentionally does NOT support:
- dynamic pricing,
- multi-chain payments,
- autonomous AI wallets,
- decentralized execution,
- cryptographic capability tokens,
- token streaming,
- advanced analytics,
- SLA guarantees,
- enterprise compliance.

---

# 19. Future Compatibility Considerations

Although excluded from MVP, architecture SHOULD remain extensible for:

### AI-Agent Wallets
### Delegated Spending
### Capability-Based Auth
### Streaming Billing
### Marketplace Discovery
### Machine-Readable Pricing
### Multi-Provider Routing

Avoid tightly coupling:
- identities,
- wallets,
- API keys,
- and settlement logic.

---

# 20. Final Technical Priorities

The MVP should prioritize:

1. ledger correctness,
2. accurate metering,
3. low operational complexity,
4. reliable settlement,
5. excellent developer experience,
6. extensible architecture.

Premature optimization should be avoided.

The MVP's primary purpose is validating:

> whether developers want programmable, usage-based API monetization infrastructure.

