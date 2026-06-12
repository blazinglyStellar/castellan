-- +goose Up

CREATE TYPE api_key_status AS ENUM ('active', 'revoked', 'expired');
CREATE TYPE provider_status AS ENUM ('active', 'inactive', 'suspended');
CREATE TYPE endpoint_status AS ENUM ('active', 'inactive', 'draft');
CREATE TYPE currency AS ENUM ('XLM', 'USDC');
CREATE TYPE entry_type AS ENUM ('deposit', 'reservation', 'deduction', 'refund', 'settlement');
CREATE TYPE ledger_status AS ENUM ('pending', 'completed', 'failed', 'cancelled');
CREATE TYPE usage_status AS ENUM ('pending', 'reserved', 'completed', 'refunded', 'failed');
CREATE TYPE batch_status AS ENUM ('pending', 'processing', 'completed', 'failed');
CREATE TYPE settlement_entry_status AS ENUM ('pending', 'completed', 'failed');
CREATE TYPE deposit_status AS ENUM ('pending', 'confirmed', 'failed');

CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email                   TEXT NOT NULL UNIQUE,
    deposit_memo            TEXT UNIQUE,
    payout_stellar_address  TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_deposit_memo ON users (deposit_memo);

CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash    TEXT NOT NULL,
    label       TEXT,
    status      api_key_status NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_user ON api_keys (user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX idx_api_keys_status ON api_keys (status);

CREATE TABLE providers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    status      provider_status NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_providers_owner ON providers (owner_id);
CREATE INDEX idx_providers_status ON providers (status);

CREATE TABLE api_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id     UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    route           TEXT NOT NULL,
    method          TEXT NOT NULL DEFAULT 'GET',
    price_amount    NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    rate_limit      INT,
    status          endpoint_status NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unique_provider_route_method UNIQUE (provider_id, route, method)
);

CREATE INDEX idx_endpoints_provider ON api_endpoints (provider_id);
CREATE INDEX idx_endpoints_status ON api_endpoints (status);

CREATE TABLE accounts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance     NUMERIC(20,10) NOT NULL DEFAULT 0,
    currency    currency NOT NULL DEFAULT 'XLM',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_accounts_owner ON accounts (owner_id);

CREATE TABLE ledger_entries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id    UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    entry_type    entry_type NOT NULL,
    amount        NUMERIC(20,10) NOT NULL,
    balance_after NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    reference_id  UUID,
    reference_type TEXT,
    status        ledger_status NOT NULL DEFAULT 'completed',
    description   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_account ON ledger_entries (account_id);
CREATE INDEX idx_ledger_type ON ledger_entries (entry_type);
CREATE INDEX idx_ledger_reference ON ledger_entries (reference_id);
CREATE INDEX idx_ledger_created ON ledger_entries (created_at);

CREATE TABLE usage_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consumer_id   UUID NOT NULL REFERENCES users(id),
    provider_id   UUID NOT NULL REFERENCES providers(id),
    endpoint_id   UUID NOT NULL REFERENCES api_endpoints(id),
    request_cost  NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    status_code   INT,
    latency_ms    INT,
    response_size INT,
    request_id    TEXT NOT NULL UNIQUE,
    status        usage_status NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_usage_consumer ON usage_events (consumer_id);
CREATE INDEX idx_usage_provider ON usage_events (provider_id);
CREATE INDEX idx_usage_endpoint ON usage_events (endpoint_id);
CREATE INDEX idx_usage_request_id ON usage_events (request_id);
CREATE INDEX idx_usage_created ON usage_events (created_at);
CREATE INDEX idx_usage_status ON usage_events (status);

CREATE TABLE deposits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    from_address    TEXT NOT NULL,
    amount          NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    memo            TEXT,
    tx_hash         TEXT NOT NULL UNIQUE,
    status          deposit_status NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    confirmed_at    TIMESTAMPTZ
);

CREATE INDEX idx_deposits_account ON deposits (account_id);
CREATE INDEX idx_deposits_tx_hash ON deposits (tx_hash);
CREATE INDEX idx_deposits_status ON deposits (status);

CREATE TABLE settlement_batches (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status        batch_status NOT NULL DEFAULT 'pending',
    total_amount  NUMERIC(20,10) NOT NULL,
    currency      currency NOT NULL DEFAULT 'XLM',
    entry_count   INT NOT NULL DEFAULT 0,
    tx_hash       TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at  TIMESTAMPTZ
);

CREATE INDEX idx_settlement_batches_status ON settlement_batches (status);
CREATE INDEX idx_settlement_batches_created ON settlement_batches (created_at);

CREATE TABLE settlement_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id        UUID NOT NULL REFERENCES settlement_batches(id) ON DELETE CASCADE,
    provider_id     UUID NOT NULL REFERENCES providers(id),
    amount          NUMERIC(20,10) NOT NULL,
    currency        currency NOT NULL DEFAULT 'XLM',
    wallet_address  TEXT NOT NULL,
    status          settlement_entry_status NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_settlement_entries_batch ON settlement_entries (batch_id);
CREATE INDEX idx_settlement_entries_provider ON settlement_entries (provider_id);
CREATE INDEX idx_settlement_entries_status ON settlement_entries (status);

-- +goose Down

DROP TABLE IF EXISTS settlement_entries;
DROP TABLE IF EXISTS settlement_batches;
DROP TABLE IF EXISTS deposits;
DROP TABLE IF EXISTS usage_events;
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS api_endpoints;
DROP TABLE IF EXISTS providers;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS deposit_status;
DROP TYPE IF EXISTS settlement_entry_status;
DROP TYPE IF EXISTS batch_status;
DROP TYPE IF EXISTS usage_status;
DROP TYPE IF EXISTS ledger_status;
DROP TYPE IF EXISTS entry_type;
DROP TYPE IF EXISTS currency;
DROP TYPE IF EXISTS endpoint_status;
DROP TYPE IF EXISTS provider_status;
DROP TYPE IF EXISTS api_key_status;
