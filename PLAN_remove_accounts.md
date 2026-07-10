# RFC: Remove `accounts` Table

The `accounts` table (columns: `id, owner_id, balance, currency`) is already 1:1 with users via `owner_id UNIQUE`. Every lookup resolves by `owner_id` — the account ID is never used independently. Merging `balance`/`currency` into `users` eliminates an unnecessary join on every API request path.

## Scope

**Tables that FK to `accounts`:**
| Table | Column | New FK target |
|---|---|---|
| `ledger_entries` | `account_id` | → `user_id` on `users` |
| `deposits` | `account_id` | → `user_id` on `users` |

**Move to `users`:** `balance NUMERIC(20,10) NOT NULL DEFAULT 0`, `currency currency NOT NULL DEFAULT 'XLM'`, `account_updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

## Migration

New file `migrations/009_flatten_accounts.sql`:

- Add `balance`, `currency`, `account_updated_at` to `users`
- Migrate data: `UPDATE users SET balance = a.balance, currency = a.currency, account_updated_at = a.updated_at FROM accounts a WHERE a.owner_id = users.id`
- Rename `ledger_entries.account_id` → `user_id`, update FK to `users(id)`
- Rename `deposits.account_id` → `user_id`, update FK to `users(id)`
- Drop `accounts` table

## SQL Query Changes

### Delete (2 files)
- `internal/repository/query/accounts.sql` — entire file (5 queries gone)
- `internal/repository/query/getAccountBalaance.sql` — entire file

### Modify (4 files)
| File | Change |
|---|---|
| `api_keys.sql` | Drop JOIN accounts; read `u.balance`, `u.currency` directly from users |
| `ledger_entries.sql` | `account_id` → `user_id`; drop JOIN accounts in `GetLedgerEntryByIDAndOwner` |
| `usage_dashboard.sql` | `account_id = $1` → `user_id = $1`; `GetUnsettledEarningsByProvider` subquery uses `users` |
| `deposits.sql` | `account_id` → `user_id` |

Run `sqlc generate` to regenerate all Go code in `internal/repository/db/`.

## Go Source Changes

### Ledger — `internal/ledger/repository.go`
- `GetAccountForUpdate(ownerID)` → `SELECT ... FROM users WHERE id = $1 FOR UPDATE`
- `UpdateAccountBalance(id)` → `UPDATE users SET balance = $2, account_updated_at = now() WHERE id = $1`
- Remove all `GetOrCreateAccount` calls (users always exist)

### Deposit — `internal/deposit/credit.go`
- Remove `GetOrCreateAccount(userID)` call
- Replace `account.ID` references with `userID` directly
- Replace `GetAccountForUpdate(ownerID)` with direct user row lock
- Replace `UpdateAccountBalance(ID: account.ID, ...)` with `UPDATE users SET balance WHERE id = $1`

### Accounts handler — `internal/accounts/service.go`
- Replace `GetAccountByOwnerID(ownerID)` with `SELECT id, balance, currency, created_at, ... FROM users WHERE id = $1`
- `GetActiveReservationsSum(accountID)` → `GetActiveReservationsSum(userID)`
- `GetLedgerEntryByIDAndOwner` query drops the accounts JOIN

### Server wiring — `internal/server/server.go`
- Remove `GetOrCreateAccount` call from `BalanceCheckerFunc` (user row always exists)

### Seed — `internal/seed/accounts.go`, `internal/seed/ledger.go`
- `INSERT INTO accounts` → `UPDATE users SET balance = ...`
- `INSERT INTO deposits` uses `user_id`
- `INSERT INTO ledger_entries` uses `user_id`

### Scripts — `scripts/seed_pokemon_usage.sql`, `scripts/seed_settlements.sql`
- `UPDATE accounts` → `UPDATE users`
- `SELECT id FROM accounts WHERE owner_id` → `SELECT id FROM users`

## Test Files (10)

Each with inline `CREATE TABLE accounts` and inline queries:
- `internal/deposit/deposit_test.go`
- `internal/deposit/credit_test.go`
- `internal/gateway/gateway_test.go`
- `internal/ledger/ledger_test.go`
- `internal/settlement/aggregator_test.go`
- `internal/settlement/reconciler_test.go`
- `internal/settlement/settlement_test.go`
- `internal/server/routes_test.go`
- `internal/dashboard/dashboard_contract_test.go`

Changes per test: merge account columns into inline `CREATE TABLE users`, change `account_id` → `user_id`, update mock signatures.

## Risk Assessment

**Benefit:** One less join per API request (balance check, reservation, deposit). Simpler mental model — no unnecessary abstraction layer.

**Risk:** The ledger transaction logic (`ReserveBalance`, `ReleaseReservation` in `repository.go`) is the most sensitive code. It uses `SELECT FOR UPDATE` and `UpdateAccountBalance` inside a `pgx.BeginTx` transaction. These need careful manual review.

**Approach:** Mechanical find-and-replace first (`account_id` → `user_id`, remove JOIN accounts), then rewire the ledger transaction queries, then fix tests. Run `make test` and `make itest` after each logical chunk.
