# Castellan MVP — Entity Relationship Diagram

```mermaid
---
id: a9755666-f0c3-49f2-a5e7-2f755340bc26
---
erDiagram
    users {
        uuid id PK
        text email UK
        text role
        text deposit_memo UK
        text payout_stellar_address
        timestamptz created_at
        timestamptz updated_at
    }

    api_keys {
        uuid id PK
        uuid user_id FK
        text key_hash
        text label
        text status
        timestamptz created_at
        timestamptz expires_at
    }

    providers {
        uuid id PK
        uuid owner_id FK
        text name
        text base_url
        text status
        timestamptz created_at
        timestamptz updated_at
    }

    api_endpoints {
        uuid id PK
        uuid provider_id FK
        text route
        text method
        numeric price_amount
        text currency
        int rate_limit
        text status
        timestamptz created_at
        timestamptz updated_at
    }

    accounts {
        uuid id PK
        uuid owner_id FK
        numeric balance
        text currency
        timestamptz created_at
        timestamptz updated_at
    }

    ledger_entries {
        uuid id PK
        uuid account_id FK
        text entry_type
        numeric amount
        numeric balance_after
        text currency
        uuid reference_id
        text reference_type
        text status
        text description
        timestamptz created_at
    }

    usage_events {
        uuid id PK
        uuid consumer_id FK
        uuid provider_id FK
        uuid endpoint_id FK
        numeric request_cost
        text currency
        int status_code
        int latency_ms
        int response_size
        text request_id
        text status
        timestamptz created_at
    }

    deposits {
        uuid id PK
        uuid account_id FK
        text from_address
        numeric amount
        text currency
        text memo
        text tx_hash
        text status
        timestamptz created_at
        timestamptz confirmed_at
    }

    settlement_batches {
        uuid id PK
        text status
        numeric total_amount
        text currency
        int entry_count
        text tx_hash
        timestamptz created_at
        timestamptz completed_at
    }

    settlement_entries {
        uuid id PK
        uuid batch_id FK
        uuid provider_id FK
        numeric amount
        text currency
        text wallet_address
        text status
        timestamptz created_at
    }

    users ||--o{ api_keys : "has"
    users ||--o{ providers : "owns"
    users ||--o| accounts : "has"
    users ||--o{ usage_events : "consumes"
    providers ||--o{ api_endpoints : "exposes"
    providers ||--o{ usage_events : "metered"
    providers ||--o{ settlement_entries : "payouts"
    accounts ||--o{ ledger_entries : "audit"
    accounts ||--o{ deposits : "funds"
    api_endpoints ||--o{ usage_events : "billed"
    settlement_batches ||--o{ settlement_entries : "contains"
```

---

## Cardinality Reference

| Relationship | Type | Meaning |
|---|---|---|
| `users \|\|--o{ api_keys` | 1:N | One user, many API keys |
| `users \|\|--o{ providers` | 1:N | One user, many providers |
| `users \|\|--o\| accounts` | 1:1 | One user, one account |
| `users \|\|--o{ usage_events` | 1:N | One consumer, many usage events |
| `providers \|\|--o{ api_endpoints` | 1:N | One provider, many endpoints |
| `providers \|\|--o{ usage_events` | 1:N | One provider, many usage events |
| `providers \|\|--o{ settlement_entries` | 1:N | One provider, many payout entries |
| `accounts \|\|--o{ ledger_entries` | 1:N | One account, many ledger entries |
| `accounts \|\|--o{ deposits` | 1:N | One account, many deposits |
| `api_endpoints \|\|--o{ usage_events` | 1:N | One endpoint, many usage events |
| `settlement_batches \|\|--o{ settlement_entries` | 1:N | One batch, many settlement entries |

---

## Table Groupings

```
Authentication:     users → api_keys
Gateway Config:     users → providers → api_endpoints
Finance (Internal): users → accounts → ledger_entries, deposits
Finance (On-chain): users.deposit_memo + users.payout_stellar_address (inlined)
Metering:           users + providers + api_endpoints → usage_events
Settlement:         settlement_batches → settlement_entries → providers
```
