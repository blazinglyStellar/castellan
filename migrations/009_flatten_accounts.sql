-- +goose Up
-- Move balance/currency from accounts into users (1:1 relationship), drop accounts.

ALTER TABLE users
    ADD COLUMN balance     NUMERIC(20,10) NOT NULL DEFAULT 0,
    ADD COLUMN currency    currency       NOT NULL DEFAULT 'XLM',
    ADD COLUMN account_updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE users SET
    balance             = a.balance,
    currency            = a.currency,
    account_updated_at  = a.updated_at
FROM accounts a
WHERE a.owner_id = users.id;

ALTER TABLE ledger_entries RENAME COLUMN account_id TO user_id;
ALTER TABLE deposits RENAME COLUMN account_id TO user_id;

ALTER TABLE ledger_entries
    DROP CONSTRAINT IF EXISTS ledger_entries_account_id_fkey,
    ADD CONSTRAINT fk_ledger_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE deposits
    DROP CONSTRAINT IF EXISTS deposits_account_id_fkey,
    ADD CONSTRAINT fk_deposit_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

DROP TABLE accounts;

-- +goose Down
CREATE TABLE accounts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance     NUMERIC(20,10) NOT NULL DEFAULT 0,
    currency    currency NOT NULL DEFAULT 'XLM',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO accounts (owner_id, balance, currency, created_at, updated_at)
SELECT id, balance, currency, now(), account_updated_at FROM users;

CREATE INDEX idx_accounts_owner ON accounts (owner_id);

ALTER TABLE ledger_entries RENAME COLUMN user_id TO account_id;
ALTER TABLE deposits RENAME COLUMN user_id TO account_id;

ALTER TABLE ledger_entries
    DROP CONSTRAINT IF EXISTS fk_ledger_user,
    ADD CONSTRAINT ledger_entries_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;

ALTER TABLE deposits
    DROP CONSTRAINT IF EXISTS fk_deposit_user,
    ADD CONSTRAINT deposits_account_id_fkey FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;

ALTER TABLE users DROP COLUMN account_updated_at, DROP COLUMN currency, DROP COLUMN balance;
