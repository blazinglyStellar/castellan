# PRD — Machine-Payable API Gateway Infrastructure

## Project Working Title
Castellan (working name)

---

# 1. Executive Summary

Castellan is a developer-first infrastructure platform that enables usage-based API monetization through programmable micropayments and metered access.

The system allows API providers to:
- monetize APIs per request,
- meter usage automatically,
- receive programmable payouts,
- support machine-to-machine transactions,
- and eventually power AI-agent economies.

Unlike traditional subscription billing systems, Castellan focuses on:
- real-time usage accounting,
- automated charging,
- low-friction developer integration,
- and blockchain-backed settlement using the Stellar ecosystem.

The initial focus is:

> A lightweight API gateway that wraps existing APIs and enables per-request monetization.

Long-term, the platform evolves into:

> programmable economic middleware for software systems and AI agents.

---

# 2. Vision Statement

Enable software systems, APIs, and AI agents to transact economically with minimal friction through programmable, usage-based payment infrastructure.

---

# 3. Problem Statement

## Current API Monetization Problems

Most API providers rely on:
- monthly subscriptions,
- API keys,
- Stripe billing,
- manual invoicing,
- enterprise contracts,
- or unsustainable free tiers.

These models create several issues:

### For API Providers
- difficult billing management,
- Stripe overhead,
- abuse from free users,
- poor monetization for low-volume APIs,
- inability to monetize small utilities,
- payment restrictions in emerging markets.

### For API Consumers
- overpaying for unused plans,
- onboarding friction,
- unnecessary subscriptions,
- inability to pay proportionally to usage,
- fragmented billing systems.

### For AI Systems
- APIs are not machine-payable,
- autonomous payment flows are difficult,
- tool usage billing lacks standardization,
- no native infrastructure for economic agents.

---

# 4. Product Goals

## Primary Goals

### G1 — Developer-Friendly API Monetization
Enable developers to monetize APIs with minimal setup.

### G2 — Usage-Based Billing
Support per-request metering and charging.

### G3 — Low Operational Overhead
Avoid requiring providers to implement:
- billing systems,
- subscription management,
- payment reconciliation.

### G4 — Stellar-Based Settlement
Leverage Stellar for:
- low-cost transfers,
- fast settlement,
- stablecoin compatibility.

### G5 — Future AI-Agent Compatibility
Design architecture that supports:
- autonomous agents,
- delegated spending,
- programmable wallets,
- machine-to-machine commerce.

---

# 5. Non-Goals (Initial Phases)

The platform will NOT initially support:
- decentralized governance,
- on-chain execution per request,
- complex DeFi integrations,
- NFT/token ecosystems,
- speculative tokenomics,
- consumer paywalls,
- creator tipping platforms,
- fiat on-ramping,
- cross-chain settlement.

---

# 6. Target Users

## Primary Users

### Independent API Developers
Examples:
- weather APIs,
- translation APIs,
- image-processing APIs,
- scraping APIs,
- analytics APIs.

### OSS Maintainers
Maintainers exposing APIs/services.

### AI Infrastructure Builders
Developers building:
- agents,
- tool ecosystems,
- autonomous workflows.

---

# 7. Core Product Concept

Castellan acts as a programmable gateway layer positioned in front of APIs.

Core workflow:

Client → Castellan → Provider API

Castellan responsibilities:
- authentication,
- usage metering,
- pricing enforcement,
- balance verification,
- settlement accounting,
- analytics.

---

# 8. High-Level Architecture

## Core Components

### 8.1 Gateway Service
Primary Go service.

Responsibilities:
- reverse proxying,
- request interception,
- authentication,
- authorization,
- pricing enforcement,
- usage recording,
- rate limiting.

### 8.2 Metering Engine
Tracks:
- requests,
- response sizes,
- compute units,
- endpoint-specific pricing.

### 8.3 Wallet & Ledger Service
Manages:
- balances,
- internal accounting,
- prepaid credits,
- transaction history.

### 8.4 Settlement Service
Handles:
- Stellar integration,
- payout batching,
- provider settlements,
- ledger reconciliation.

### 8.5 SDK Layer
Developer SDKs for:
- JavaScript/TypeScript,
- Go,
- Python.

### 8.6 Dashboard
Web dashboard for:
- API providers,
- consumers,
- usage analytics,
- billing visibility.

---

# 9. Recommended Technology Stack

## Backend (Primary)

### Golang
Primary language.

Reasons:
- excellent concurrency model,
- ideal for gateways/proxies,
- strong networking performance,
- lightweight deployments,
- production-grade infrastructure tooling ecosystem.

Recommended libraries:
- Gin or Chi
- gRPC
- Zap logger
- Viper
- Cobra CLI
- Prometheus client

---

## Database Layer

### PostgreSQL
Primary relational database.

Stores:
- users,
- wallets,
- balances,
- pricing configs,
- usage records,
- settlement logs.

### Redis
Caching and temporary state.

Uses:
- rate limiting,
- session state,
- pricing cache,
- temporary reservations.

---

## Infrastructure

### Docker
Containerization.

### Kubernetes (Future)
Scalable orchestration.

### NGINX / Envoy (Optional)
Edge proxy integration.

---

## Blockchain Layer

### Stellar SDKs
Settlement integration.

Potential usage:
- XLM transfers,
- USDC on Stellar,
- batched payouts,
- wallet generation.

---

## Frontend

### Next.js
Dashboard frontend.

### Tailwind CSS
UI styling.

---

## Observability

### Prometheus
Metrics.

### Grafana
Monitoring dashboards.

### OpenTelemetry
Distributed tracing.

---

# 10. Data Model Overview

## Core Entities

### User
- id
- email
- wallet_id
- role
- created_at

### Wallet
- id
- owner_id
- balance
- currency
- settlement_address

### API Provider
- id
- owner_id
- base_url
- pricing_model
- status

### API Endpoint
- id
- provider_id
- route
- price_per_request
- pricing_type

### Usage Event
- id
- consumer_id
- endpoint_id
- request_cost
- timestamp
- status

### Settlement Batch
- id
- total_amount
- tx_hash
- status
- created_at

---

# 11. Authentication & Authorization

## Initial Authentication

### API Keys
Simple developer onboarding.

### Session Tokens
Temporary scoped access.

---

## Future Authentication

### Wallet Signatures
Cryptographic request authorization.

### Capability Tokens
Scoped permissions.

Example:

```json
{
  "max_daily_spend": 5,
  "allowed_apis": ["weather", "search"],
  "expires_at": 123456789
}
```

---

# 12. Payment Model

## Initial Model — Prepaid Balances

Flow:
1. User deposits funds.
2. Balance credited internally.
3. Requests deduct credits.
4. Providers settled periodically.

Advantages:
- low latency,
- simpler architecture,
- reduced chain interactions.

---

# 13. Settlement Strategy

## Batched Settlement

Do NOT settle every request individually.

Instead:
- aggregate usage internally,
- periodically execute settlement batches.

Benefits:
- lower fees,
- higher throughput,
- reduced blockchain dependency.

---

# 14. Pricing Models

## Phase 1

### Fixed Per Request
Examples:
- /search → 0.0001 XLM
- /image → 0.02 XLM

---

## Future Pricing Models

### Per Compute Unit
### Per Token
### Per MB
### Dynamic Load Pricing
### Subscription Hybrid

---

# 15. Security Requirements

## Core Security Measures

### Replay Protection
Prevent request reuse.

### Rate Limiting
Per-user and per-endpoint controls.

### Request Signing
Future cryptographic verification.

### Usage Validation
Prevent fraudulent billing.

### Scoped Permissions
Prevent unrestricted wallet spending.

---

# 16. Developer Experience Goals

## Primary DX Principles

### Minimal Integration
Target:

```bash
castellan wrap https://myapi.com
```

### Drop-In Proxying
No API rewrites required.

### Simple SDKs
Minimal setup.

### Transparent Billing
Clear usage visibility.

---

# 17. API Gateway Features

## Phase 1 Features

### Request Metering
### Fixed Pricing
### Reverse Proxying
### API Key Auth
### Usage Logs
### Provider Dashboard

---

## Phase 2 Features

### Stellar Wallet Integration
### Real Balances
### Settlement Batching
### Rate Limiting
### Webhooks

---

## Phase 3 Features

### Multi-Tenant Marketplace
### SDK Ecosystem
### Usage Analytics
### CLI Tools
### OpenAPI Integration

---

## Phase 4 Features

### Agent Wallets
### Delegated Spending
### Capability-Based Auth
### Streaming Payments
### Autonomous Billing Policies

---

# 18. AI-Agent Compatibility Roadmap

## Design Principles

The system should treat:
- humans,
- scripts,
- APIs,
- AI agents

as programmable economic actors.

---

## Required Future Capabilities

### Delegated Wallets
Human-controlled parent wallets authorize agents.

### Spending Policies
Examples:
- max daily spend,
- approved tools,
- budget caps.

### Session-Based Billing
Persistent payment sessions.

### Machine-Readable Pricing
Agents discover costs programmatically.

### Tool Discovery Layer
Agents discover payable APIs/tools.

---

# 19. Suggested Development Phases

# Phase 1 — Core Gateway MVP

## Goal
Validate core usage-based billing concept.

## Deliverables
- Go reverse proxy gateway
- request metering
- fixed pricing rules
- in-memory ledger
- API key auth
- usage logging
- minimal dashboard

## Technologies
- Go
- PostgreSQL
- Redis
- Docker

## Success Criteria
- API provider can wrap an API
- requests are metered
- balances deduct correctly
- dashboard reflects usage

---

# Phase 2 — Real Payment Infrastructure

## Goal
Integrate real settlement.

## Deliverables
- Stellar wallet integration
- prepaid balances
- batched settlement
- provider payouts
- transaction history

## Additional Technologies
- Stellar SDK
- queue workers

## Success Criteria
- users can deposit funds
- providers receive payouts
- settlements reconcile correctly

---

# Phase 3 — Developer Platform

## Goal
Improve adoption and usability.

## Deliverables
- JS SDK
- Go SDK
- provider onboarding
- dashboard analytics
- pricing management
- CLI tooling
- OpenAPI support

## Success Criteria
- third-party developers integrate independently
- APIs onboard with minimal support

---

# Phase 4 — Marketplace & Ecosystem

## Goal
Create API economy infrastructure.

## Deliverables
- API discovery
- searchable marketplace
- usage reputation
- provider profiles
- OSS funding integrations

## Success Criteria
- multiple APIs using platform
- discoverability ecosystem exists

---

# Phase 5 — AI-Agent Infrastructure

## Goal
Enable autonomous software commerce.

## Deliverables
- agent wallets
- delegated spending
- capability tokens
- programmable policies
- session-based payment streams
- machine-readable pricing schemas

## Success Criteria
- AI agents autonomously consume APIs
- spending constraints enforced
- high-frequency usage supported

---

# 20. Open Source Strategy

## Initial License

Recommended:
- Apache 2.0
or
- MIT

---

## Open Source Goals

### Encourage Ecosystem Contributions
### Build Stellar Developer Tooling
### Enable OSS Sustainability
### Attract Maintainers and Integrators

---

# 21. Potential OSS Modules

Possible separate repositories:
- gateway core,
- JS SDK,
- Go SDK,
- settlement service,
- dashboard,
- CLI,
- provider examples.

---

# 22. Long-Term Strategic Positioning

Castellan should evolve toward:

> programmable economic middleware for software systems.

Potential long-term positioning:
- machine-payable APIs,
- AI-agent commerce infrastructure,
- programmable billing systems,
- OSS monetization infrastructure,
- usage-based economic networks.

---

# 23. Competitive Landscape

## Comparable Systems

### API Gateways
- Kong
- Tyk
- AWS API Gateway

### Payment Infrastructure
- Stripe Billing
- Coinbase Commerce
- Superfluid

### Blockchain Infrastructure
- Lightning-based API monetization
- L402 ecosystems

---

# 24. Key Differentiators

## Developer-First UX
Minimal setup.

## Usage-Based Billing
Granular monetization.

## Blockchain Settlement
Low-cost programmable payouts.

## AI-Agent Readiness
Architected for autonomous systems.

## OSS Alignment
Supports sustainable API ecosystems.

---

# 25. Risks & Challenges

## Technical Risks

### Latency Overhead
Gateway must remain performant.

### Fraud & Abuse
Need robust replay/rate protections.

### Pricing Complexity
Avoid over-engineering early.

### Blockchain Dependency
Need resilient fallback handling.

---

## Product Risks

### Developer Adoption
DX quality is critical.

### Market Education
Need clear positioning.

### Premature AI Focus
Avoid over-optimizing for speculative workflows.

---

# 26. Success Metrics

## Technical Metrics
- gateway latency,
- settlement accuracy,
- throughput,
- uptime.

## Product Metrics
- APIs onboarded,
- monthly requests,
- total payment volume,
- active developers,
- SDK adoption.

---

# 27. Recommended Initial Repository Structure

```text
castellan/
├── cmd/
│   ├── gateway/
│   ├── worker/
│   └── cli/
├── internal/
│   ├── auth/
│   ├── billing/
│   ├── gateway/
│   ├── ledger/
│   ├── settlement/
│   ├── pricing/
│   ├── metering/
│   ├── wallet/
│   └── storage/
├── sdk/
│   ├── js/
│   ├── go/
│   └── python/
├── deployments/
├── migrations/
├── docs/
└── examples/
```

---

# 28. Recommended MVP Priorities

Highest priority:
1. gateway stability,
2. accurate metering,
3. pricing engine,
4. prepaid balances,
5. developer onboarding simplicity.

Lower priority:
- advanced cryptography,
- decentralized governance,
- tokenization,
- AI marketplaces.

---

# 29. Recommended Initial Positioning

## Primary Positioning

> Usage-based API monetization infrastructure.

---

## Secondary Positioning

> Machine-payable middleware for APIs and AI systems.

---

# 30. Final Notes

The strongest path for Castellan is not competing with existing payment processors directly.

Instead, it should focus on:
- programmable billing,
- machine-native payments,
- usage-based infrastructure,
- and developer-first monetization workflows.

The project should prioritize:
- simplicity,
- reliability,
- extensibility,
- and excellent developer experience.

The long-term opportunity lies in becoming foundational infrastructure for:
- autonomous software systems,
- AI agents,
- and sustainable API economies.

