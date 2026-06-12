# Castellan MVP Database Schema (PostgreSQL)

Derived from MVP PRD (Phase 1 + Phase 2)

---

## Enums

```sql
CREATE TYPE user_role AS ENUM ('provider', 'consumer', 'both', 'admin');
CREATE TYPE api_key_status AS ENUM ('active', 'revoked', 'expired');
CREATE TYPE provider_status AS ENUM ('active', 'inactive', 'suspended');
CREATE TYPE endpoint_status AS ENUM ('active', 'inactive');
CREATE TYPE currency AS ENUM ('XLM', 'USDC');
CREATE TYPE entry_type AS ENUM ('deposit', 'reservation', 'deduction', 'refund', 'settlement');
CREATE TYPE ledger_status AS ENUM ('pending', 'completed', 'failed', 'cancelled');
CREATE TYPE usage_status AS ENUM ('pending', 'reserved', 'completed', 'refunded', 'failed');
CREATE TYPE batch_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE settlement_entry_status AS ENUM ('pending', 'completed', 'failed');
CREATE TYPE deposit_status AS ENUM ('pending', 'confirmed', 'failed');
```

---

## Tables

### 1. users

Core identity for both providers and consumers.

```sql
CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                   TEXT NOT NULL UNIQUE,
    role                    user_role NOT NULL DEFAULT 'consumer',

    -- Stellar deposit routing: assigned on first deposit request, used by deposit watcher
    deposit_memo            TEXT UNIQUE,

    -- Stellar payout destination: set by provider in dashboard to receive settlements
    payout_stellar_address  TEXT,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_role ON users (role);
CREATE INDEX idx_users_deposit_memo ON users (deposit_memo);
```

---

### 2. api_keys

Bearer token authentication.

```sql
CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash    TEXT NOT NULL,                     -- SHA-256 or bcrypt hash of ca_xxx key
    label       TEXT,                              -- human-readable name (e.g. "prod", "dev")
    status      api_key_status NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_user ON api_keys (user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX idx_api_keys_status ON api_keys (status);
```

---

### 3. providers

Upstream API provider configurations.

```sql
CREATE TABLE providers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,                     -- upstream API base URL
    status      provider_status NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_providers_owner ON providers (owner_id);
CREATE INDEX idx_providers_status ON providers (status);
```

---

### 4. api_endpoints

Individual routes with fixed per-request pricing.

```sql
CREATE TABLE api_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id     UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    route           TEXT NOT NULL,                 -- e.g. /search, /weather
    method          TEXT NOT NULL DEFAULT 'GET',   -- GET, POST, PUT, DELETE, ANY
    price_amount    NUMERIC(20,10) NOT NULL,       -- e.g. 0.0001
    currency        currency NOT NULL DEFAULT 'XLM',
    rate_limit      INT,                           -- requests per minute (NULL = unlimited)
    status          endpoint_status NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT unique_provider_route_method UNIQUE (provider_id, route, method)
);

CREATE INDEX idx_endpoints_provider ON api_endpoints (provider_id);
CREATE INDEX idx_endpoints_status ON api_endpoints (status);
```

---

### 5. accounts

Internal prepaid credit balance. This is the DB's own record of how much credit a user has — **not** the on-chain Stellar balance. The gateway reads this for fast authorization without querying Stellar per request.

```sql
CREATE TABLE accounts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance     NUMERIC(20,10) NOT NULL DEFAULT 0,
    currency    currency NOT NULL DEFAULT 'XLM',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_accounts_owner ON accounts (owner_id);
```

---

### 6. ledger_entries

Immutable audit trail for all internal credit operations.

```sql
CREATE TABLE ledger_entries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    entry_type    entry_type NOT NULL,
    amount        NUMERIC(20,10) NOT NULL,         -- positive for credits, negative for debits
    balance_after NUMERIC(20,10) NOT NULL,         -- snapshot of account balance after entry
    currency      currency NOT NULL DEFAULT 'XLM',
    reference_id  UUID,                            -- FK to usage_events, settlement_batches, or deposits
    reference_type TEXT,                           -- 'usage_event', 'settlement_batch', 'deposit'
    status        ledger_status NOT NULL DEFAULT 'completed',
    description   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_account ON ledger_entries (account_id);
CREATE INDEX idx_ledger_type ON ledger_entries (entry_type);
CREATE INDEX idx_ledger_reference ON ledger_entries (reference_id);
CREATE INDEX idx_ledger_created ON ledger_entries (created_at);
```

---

### 7. usage_events

Per-request metering records.

```sql
CREATE TABLE usage_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consumer_id   UUID NOT NULL REFERENCES users(id),
    provider_id   UUID NOT NULL REFERENCES providers(id),
    endpoint_id   UUID NOT NULL REFERENCES api_endpoints(id),
    request_cost  NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    status_code   INT,                              -- upstream HTTP status code
    latency_ms    INT,
    response_size INT,                              -- upstream response body size in bytes
    request_id    TEXT NOT NULL,                     -- idempotency key / correlation ID
    status        usage_status NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_usage_consumer ON usage_events (consumer_id);
CREATE INDEX idx_usage_provider ON usage_events (provider_id);
CREATE INDEX idx_usage_endpoint ON usage_events (endpoint_id);
CREATE INDEX idx_usage_request_id ON usage_events (request_id);
CREATE INDEX idx_usage_created ON usage_events (created_at);
CREATE INDEX idx_usage_status ON usage_events (status);
```

---

### 8. deposits

Incoming Stellar payment tracking. The deposit watcher detects on-chain payments, creates a deposit record, and credits the associated internal account.

```sql
CREATE TABLE deposits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    from_address    TEXT NOT NULL,                  -- sender Stellar account
    amount          NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    memo            TEXT,
    tx_hash         TEXT NOT NULL,                  -- Stellar transaction hash
    status          deposit_status NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at    TIMESTAMPTZ
);

CREATE INDEX idx_deposits_account ON deposits (account_id);
CREATE INDEX idx_deposits_tx_hash ON deposits (tx_hash);
CREATE INDEX idx_deposits_status ON deposits (status);
```

---

### 9. settlement_batches

Grouped Stellar payout transactions.

```sql
CREATE TABLE settlement_batches (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status        batch_status NOT NULL DEFAULT 'pending',
    total_amount  NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    entry_count   INT NOT NULL DEFAULT 0,
    tx_hash       TEXT,                            -- Stellar transaction hash after submission
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at  TIMESTAMPTZ
);

CREATE INDEX idx_settlement_batches_status ON settlement_batches (status);
CREATE INDEX idx_settlement_batches_created ON settlement_batches (created_at);
```

---

### 10. settlement_entries

Individual provider payouts within a batch.

```sql
CREATE TABLE settlement_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id        UUID NOT NULL REFERENCES settlement_batches(id) ON DELETE CASCADE,
    provider_id     UUID NOT NULL REFERENCES providers(id),
    amount          NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    wallet_address  TEXT NOT NULL,                  -- provider's Stellar destination
    status          settlement_entry_status NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_settlement_entries_batch ON settlement_entries (batch_id);
CREATE INDEX idx_settlement_entries_provider ON settlement_entries (provider_id);
CREATE INDEX idx_settlement_entries_status ON settlement_entries (status);
```

---

## Entity-Relationship Summary

```
users
 ├── api_keys          (1:N) — authentication credentials
 ├── providers         (1:N) — APIs a user owns
 │    └── api_endpoints (1:N) — routes with pricing
 └── accounts          (1:1) — internal prepaid credit (balance in DB)
      ├── ledger_entries   (1:N) — immutable audit trail
      └── deposits         (1:N) — incoming Stellar payments

usage_events           — also references users (consumer), providers & endpoints (metering)
settlement_batches     — grouped payouts
 └── settlement_entries (1:N) — per-provider payouts
```

Note: Stellar wallet references are inlined on `users.deposit_memo` and `users.payout_stellar_address` — no separate table needed.

## Key Design Decisions

1. **Stellar references inlined on `users`** — A user needs one deposit memo (for routing incoming payments) and one payout address (for receiving settlements). A separate `stellar_wallets` table was unnecessary.
2. **`accounts` vs on-chain balance** — Internal credit (`accounts.balance`) is the gateway's source of truth for fast auth. On-chain Stellar balances are never stored in the DB.
3. **`balance_after` in ledger_entries** — Enables point-in-time balance audit without summing full history.
4. **`request_id` on usage_events** — Idempotency for retry-safe metering.
5. **`reference_id` + `reference_type` in ledger_entries** — Polymorphic FK avoids separate tables per entry type.
6. **Redis for temporary reservations** — Not persisted to PG; reservations live in Redis with TTL.
7. **`rate_limit` on api_endpoints** — Configurable per-endpoint throttling (enforced via Redis token bucket).
