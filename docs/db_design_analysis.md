# Castellan MVP — Database Design Analysis

---

## 1. Overview

Castellan's database is a PostgreSQL-backed relational model with 10 tables that support two core concerns:

1. **Gateway operations** — authentication, metering, pricing, and balance checks (fast path, per-request)
2. **Financial settlement** — deposit tracking, ledger audit trail, and batched Stellar payouts (async, background)

The database is not the source of truth for on-chain data. The Stellar network owns balances and transaction finality. The DB tracks internal prepaid credit (`accounts.balance`) that the gateway reads for sub-100ms authorization decisions.

---

## 2. Tables and Columns

---

### 2.1 `users`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK, `gen_random_uuid()` | Unique identifier |
| `email` | TEXT | NOT NULL, UNIQUE | Login/identity |
| `role` | user_role | NOT NULL, DEFAULT 'consumer' | `provider`, `consumer`, `both`, or `admin` |
| `deposit_memo` | TEXT | UNIQUE | Assigned on first deposit request; used by deposit watcher to route incoming payments |
| `payout_stellar_address` | TEXT | — | Stellar G-pubkey where provider receives settlements; set in dashboard |
| `created_at` | TIMESTAMPTZ | NOT NULL | Row creation |
| `updated_at` | TIMESTAMPTZ | NOT NULL | Row last update |

**Purpose:** Core identity. Every actor in the system — whether they own APIs (provider) or call them (consumer) — is a user. A single user can be both roles.

**Stellar references inlined:** A user needs exactly one deposit memo (for routing incoming payments) and one payout address (for receiving settlements). A separate `stellar_wallets` table was unnecessary overhead — these two columns cover both needs without joins.

---

### 2.2 `api_keys`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK | Unique identifier |
| `user_id` | UUID | FK → users(id), NOT NULL | Owning user |
| `key_hash` | TEXT | NOT NULL | SHA-256/bcrypt hash of the raw `fg_xxx` key |
| `label` | TEXT | — | Human-readable name (e.g. "prod", "dev") |
| `status` | api_key_status | NOT NULL, DEFAULT 'active' | `active`, `revoked`, `expired` |
| `created_at` | TIMESTAMPTZ | NOT NULL | Row creation |
| `expires_at` | TIMESTAMPTZ | — | Optional TTL |

**Purpose:** Bearer token auth for consumers calling proxied APIs. The raw key is never stored — only its hash. This is the same pattern GitHub and Stripe use.

**Why `label`:** Lets users manage multiple keys (e.g. rotate without downtime).

---

### 2.3 `providers`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK | Unique identifier |
| `owner_id` | UUID | FK → users(id), NOT NULL | The user who registered this API |
| `name` | TEXT | NOT NULL | Display name |
| `base_url` | TEXT | NOT NULL | Upstream API root URL |
| `status` | provider_status | NOT NULL, DEFAULT 'active' | `active`, `inactive`, `suspended` |
| `created_at` | TIMESTAMPTZ | NOT NULL | Row creation |
| `updated_at` | TIMESTAMPTZ | NOT NULL | Row last update |

**Purpose:** Represents an upstream API service. A provider wraps their API behind Castellan by registering a provider + endpoints.

**Why separate from `users`:** A user may own multiple APIs. Separating enables independent status management (suspend a provider without disabling the user) and future multi-tenant features.

---

### 2.4 `api_endpoints`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK | Unique identifier |
| `provider_id` | UUID | FK → providers(id), NOT NULL | Parent provider |
| `route` | TEXT | NOT NULL | Path pattern (e.g. `/search`) |
| `method` | TEXT | NOT NULL, DEFAULT 'GET' | HTTP method or 'ANY' |
| `price_amount` | NUMERIC(20,10) | NOT NULL | Cost per request (e.g. 0.0001) |
| `currency` | currency | NOT NULL, DEFAULT 'XLM' | Token denomination |
| `rate_limit` | INT | — | Requests/minute (NULL = unlimited) |
| `status` | endpoint_status | NOT NULL, DEFAULT 'active' | `active`, `inactive` |
| `created_at` | TIMESTAMPTZ | NOT NULL | Row creation |
| `updated_at` | TIMESTAMPTZ | NOT NULL | Row last update |

**Unique constraint:** `(provider_id, route, method)` — prevents duplicate endpoint registrations.

**Purpose:** Defines what routes exist under a provider, what they cost, and how fast they can be called.

**Why `rate_limit` here and not in Redis only:** Persisting the limit in Postgres means it survives restarts and is the source of truth. Redis enforces the rate at runtime but reads the config from this column.

---

### 2.5 `accounts`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK | Unique identifier |
| `owner_id` | UUID | FK → users(id), UNIQUE, NOT NULL | Owning user |
| `balance` | NUMERIC(20,10) | NOT NULL, DEFAULT 0 | Internal prepaid credit |
| `currency` | currency | NOT NULL, DEFAULT 'XLM' | Denomination of the balance |
| `created_at` | TIMESTAMPTZ | NOT NULL | Row creation |
| `updated_at` | TIMESTAMPTZ | NOT NULL | Row last update |

**Purpose:** The gateway's source of truth for "does this user have enough funds?" The balance is debited synchronously during request processing and credited asynchronously when deposits are confirmed.

**Why `UNIQUE` on `owner_id`:** One account per user in MVP. Simplifies balance lookups — no need to sum across multiple accounts. Can be relaxed to `(owner_id, currency)` if multi-currency emerges.

**Why this is NOT the Stellar balance:** The Stellar ledger is the source of truth for on-chain balances. This table tracks a parallel internal credit that gets updated only when deposits are confirmed or usage is deducted. This separation allows the gateway to operate at low latency without querying the Stellar network per request.

---

### 2.6 `ledger_entries`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK | Unique identifier |
| `account_id` | UUID | FK → accounts(id), NOT NULL | Account this entry affects |
| `entry_type` | entry_type | NOT NULL | `deposit`, `reservation`, `deduction`, `refund`, `settlement` |
| `amount` | NUMERIC(20,10) | NOT NULL | Positive = credit, negative = debit |
| `balance_after` | NUMERIC(20,10) | NOT NULL | Snapshot of account balance post-entry |
| `currency` | currency | NOT NULL, DEFAULT 'XLM' | Denomination |
| `reference_id` | UUID | — | Links to related record (usage_event, deposit, etc.) |
| `reference_type` | TEXT | — | Discriminator for the polymorphic reference |
| `status` | ledger_status | NOT NULL, DEFAULT 'completed' | `pending`, `completed`, `failed`, `cancelled` |
| `description` | TEXT | — | Human-readable explanation |
| `created_at` | TIMESTAMPTZ | NOT NULL | Row creation |

**Purpose:** Immutable audit trail. Every financial operation (deposit, deduction, refund, settlement) produces a ledger entry. This is the single source of truth for reconstructing account history.

**Why `balance_after`:** Enables point-in-time balance queries without summing the entire entry history. Also makes it easy to detect inconsistencies — if `balance_after` doesn't match the sum of previous entries, something is wrong.

**Why polymorphic reference (`reference_id` + `reference_type`):** A ledger entry can reference a `usage_event`, `deposit`, or `settlement_batch`. Using a polymorphic pattern avoids creating three separate nullable FK columns. The tradeoff is no declarative referential integrity — enforcement must happen in application code.

---

### 2.7 `usage_events`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK | Unique identifier |
| `consumer_id` | UUID | FK → users(id), NOT NULL | The user who made the request |
| `provider_id` | UUID | FK → providers(id), NOT NULL | The provider whose API was called |
| `endpoint_id` | UUID | FK → api_endpoints(id), NOT NULL | The specific endpoint hit |
| `request_cost` | NUMERIC(20,10) | NOT NULL | Amount deducted |
| `currency` | currency | NOT NULL, DEFAULT 'XLM' | Denomination |
| `status_code` | INT | — | Upstream HTTP response code |
| `latency_ms` | INT | — | Upstream response time |
| `response_size` | INT | — | Upstream response body size |
| `request_id` | TEXT | NOT NULL | Idempotency key / correlation ID |
| `status` | usage_status | NOT NULL, DEFAULT 'pending' | `pending`, `reserved`, `completed`, `refunded`, `failed` |
| `created_at` | TIMESTAMPTZ | NOT NULL | Row creation |

**Purpose:** Per-request metering record. Every billable proxied request generates one row. This data drives provider analytics, consumer usage logs, and the settlement aggregation query.

**Why `request_id`:** Provides idempotency. If the gateway crashes after forwarding a request but before persisting the event, it can safely retry with the same request_id and the unique constraint prevents double-billing.

**Why denormalized `provider_id` + `endpoint_id` on the same row:** Queries aggregated by provider (e.g. "total earnings for this provider") are extremely common and would otherwise require a join through `api_endpoints`. The extra column saves a frequent join at minimal storage cost.

---

### 2.8 `deposits`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK | Unique identifier |
| `account_id` | UUID | FK → accounts(id), NOT NULL | Account credited |
| `from_address` | TEXT | NOT NULL | Sender's Stellar account |
| `amount` | NUMERIC(20,10) | NOT NULL | Value received |
| `currency` | currency | NOT NULL, DEFAULT 'XLM' | Denomination |
| `memo` | TEXT | — | Stellar memo from the transaction; matched against `users.deposit_memo` for routing |
| `tx_hash` | TEXT | NOT NULL | Stellar transaction hash |
| `status` | deposit_status | NOT NULL, DEFAULT 'pending' | `pending`, `confirmed`, `failed` |
| `created_at` | TIMESTAMPTZ | NOT NULL | First detected |
| `confirmed_at` | TIMESTAMPTZ | — | When sufficient confirmations met |

**Purpose:** Tracks every incoming Stellar payment detected by the deposit watcher. When a deposit is confirmed, a corresponding `ledger_entry` of type `deposit` credits the account.

**Why `tx_hash` is unique (implicitly):** Prevents double-crediting the same on-chain transaction.

**Why `memo` is stored here but routing is on `users`:** The deposit memo from the Stellar transaction is matched against `users.deposit_memo` by the deposit watcher. Storing it on the deposit row provides an immutable record of what memo was used, enabling audit and manual recovery when the memo is missing or wrong.

---

### 2.9 `settlement_batches`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK | Unique identifier |
| `status` | batch_status | NOT NULL, DEFAULT 'pending' | `pending`, `processing`, `completed`, `failed` |
| `total_amount` | NUMERIC(20,10) | NOT NULL | Sum of all entries |
| `currency` | currency | NOT NULL, DEFAULT 'XLM' | Denomination |
| `entry_count` | INT | NOT NULL, DEFAULT 0 | Number of payouts in this batch |
| `tx_hash` | TEXT | — | Stellar transaction hash after submission |
| `created_at` | TIMESTAMPTZ | NOT NULL | Batch creation |
| `completed_at` | TIMESTAMPTZ | — | When Stellar confirmed |

**Purpose:** Groups multiple provider payouts into a single Stellar transaction (or sequential transactions). This is the core of the batched settlement strategy — instead of one Stellar tx per usage event, the settlement worker aggregates and sends bulk payouts.

---

### 2.10 `settlement_entries`

| Column | Type | Constraints | Purpose |
|---|---|---|---|
| `id` | UUID | PK | Unique identifier |
| `batch_id` | UUID | FK → settlement_batches(id), NOT NULL | Parent batch |
| `provider_id` | UUID | FK → providers(id), NOT NULL | Receiving provider |
| `amount` | NUMERIC(20,10) | NOT NULL | Payout amount |
| `currency` | currency | NOT NULL, DEFAULT 'XLM' | Denomination |
| `wallet_address` | TEXT | NOT NULL | Provider's Stellar destination |
| `status` | settlement_entry_status | NOT NULL, DEFAULT 'pending' | `pending`, `completed`, `failed` |
| `created_at` | TIMESTAMPTZ | NOT NULL | Row creation |

**Purpose:** Individual provider payout within a batch. Allows per-provider status tracking — some entries in a batch may fail while others succeed.

**Why `wallet_address` is denormalized:** At settlement time, the wallet address is copied from `users.payout_stellar_address`. This creates an immutable record of where the money was actually sent, even if the provider changes their payout address later.

---

## 3. Entity Relationships

```
users (1) ───────< (N) api_keys
users (1) ───────< (N) providers
users (1) ──────── (1) accounts
users (1) ───────< (N) usage_events          (as consumer)

providers (1) ───< (N) api_endpoints
providers (1) ───< (N) usage_events          (metered usage)
providers (1) ───< (N) settlement_entries    (payouts)

accounts (1) ────< (N) ledger_entries
accounts (1) ────< (N) deposits

api_endpoints (1) < (N) usage_events

settlement_batches (1) < (N) settlement_entries
```

### Cardinality notes

| Relationship | Type | Rationale |
|---|---|---|
| users → accounts | 1:1 | MVP simplicity. One prepaid account per user. |  
| providers → api_endpoints | 1:N | A single API service exposes many routes. |
| accounts → ledger_entries | 1:N | Many financial events per account over time. |
| settlement_batches → settlement_entries | 1:N | One batch, many provider payouts. |

Note: Stellar wallet references are inlined on `users.deposit_memo` and `users.payout_stellar_address` — no separate table or relationship.

### Orphan protection

All child tables use `ON DELETE CASCADE` from their parent. This is acceptable for MVP because the only deletion scenario is administrative cleanup. In production, soft-deletes or deletion prevention would be preferred.

---

## 4. Design Justification

### 4.1 Why `accounts` is separate from Stellar wallet data?

Earlier in the design, a single `wallets` table held both `balance` and `stellar_address`, conflating internal credit with on-chain wallet references. These were then split into `accounts` + `stellar_wallets`, which was later simplified further.

The current design inlines Stellar data on `users`:

- **`accounts`** — DB-owned internal credit, fast to read, gateway's authorization mechanism
- **`users.deposit_memo`** — opaque routing token for the deposit watcher
- **`users.payout_stellar_address`** — provider's destination for settlement payouts

This means:
- The gateway never queries Stellar during request processing
- The `accounts` table can be tuned for OLTP performance (small row size, hot cache)
- Stellar routing data is write-once, read-rarely — no reason for a separate table

### 4.2 Why `balance_after` on `ledger_entries`?

Without `balance_after`, computing a current balance requires:

```sql
SELECT SUM(amount) FROM ledger_entries WHERE account_id = $1;
```

This is O(n) in the number of entries and gets slower as the account ages. With `balance_after`, the current balance is simply the last entry's value:

```sql
SELECT balance_after FROM ledger_entries
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT 1;
```

Additionally, `balance_after` acts as a consistency check — the application can verify that `balance_after` of entry N equals `balance_after` of entry N-1 + `amount` of entry N. Mismatches indicate a bug or tampering.

### 4.3 Why `request_id` on `usage_events`?

The request lifecycle is:

1. Reserve funds (Redis)
2. Forward request upstream
3. Capture response
4. Write `usage_event` row
5. Deduct from `accounts.balance`

If the gateway crashes between steps 3 and 4, the request was served but not billed. The upstream client will retry (standard HTTP retry behavior). Without idempotency, the retry would create a duplicate `usage_event` and double-bill.

`request_id` is generated by the gateway at step 1 and included in every retry. The UNIQUE constraint on `request_id` prevents duplicate rows. This is the same pattern Stripe uses for idempotency keys.

### 4.4 Why polymorphic reference in `ledger_entries`?

A ledger entry can reference:
- A `usage_event` (when funds are deducted for a request)
- A `deposit` (when funds are credited from an incoming payment)
- A `settlement_batch` (when funds are settled to a provider)

Using `reference_id` + `reference_type` avoids three nullable FK columns:

```sql
-- Instead of:
usage_event_id     UUID REFERENCES usage_events(id),
deposit_id         UUID REFERENCES deposits(id),
settlement_batch_id UUID REFERENCES settlement_batches(id),
```

The tradeoff is lost declarative referential integrity. Application code must ensure reference_id points to a valid row of the correct type. For MVP this is acceptable; in production, a CHECK constraint with a custom function could be added.

### 4.5 Why one `accounts` table and not a full double-entry ledger?

A true double-entry ledger would have `journal_entries` (two rows per transaction: debit one account, credit another). Castellan's MVP doesn't need this because:

- The gateway only manages consumer prepaid accounts
- Provider payouts are handled through Stellar, not internal transfers
- The `accounts` table acts as a single-entry ledger with an audit trail (`ledger_entries`)

If Castellan later supports internal transfers (e.g. consumer-to-consumer), a double-entry system would become necessary.

### 4.6 Why denormalize `provider_id` on `usage_events` when `endpoint_id` already implies it?

Because the most frequent analytical query is "how much did provider X earn?":

```sql
SELECT SUM(request_cost) FROM usage_events WHERE provider_id = $1;
```

Without the denormalized column, every such query requires a join:

```sql
SELECT SUM(ue.request_cost)
FROM usage_events ue
JOIN api_endpoints ae ON ae.id = ue.endpoint_id
WHERE ae.provider_id = $1;
```

The join is cheap with proper indexing, but at Castellan's scale (potentially millions of usage events), the denormalized column eliminates it entirely. The storage cost is one UUID per row (~16 bytes).

---

## 5. Potential Problems and Risks

### 5.1 `accounts.balance` can drift from reality

**Problem:** `accounts.balance` is maintained by application code writing `ledger_entries` and updating the balance. A bug, crash between steps, or race condition could cause the balance to diverge from the sum of ledger entries.

**Mitigation:**
- `balance_after` in `ledger_entries` enables automated reconciliation — a background worker can periodically verify `balance_after = previous.balance_after + amount` for every entry
- Use `SELECT ... FOR UPDATE` when deducting balances to prevent race conditions
- Consider a procedural reconciliation worker that recomputes balances from scratch on a schedule

**Severity:** High. Financial correctness depends on this.

### 5.2 Polymorphic reference has no FK enforcement

**Problem:** `reference_id` + `reference_type` cannot be enforced at the database level. Application bugs could create ledger entries pointing to non-existent or wrong-type records.

**Mitigation:**
- Add application-level validation on write
- Consider PostgreSQL triggers or a CHECK constraint with a helper function in production
- Or revert to separate nullable FK columns if bugs emerge

**Severity:** Medium. Hard to catch early since nothing fails at INSERT time.

### 5.3 `ON DELETE CASCADE` is dangerous in production

**Problem:** All child tables use `ON DELETE CASCADE`. Accidentally deleting a user would cascade-delete their API keys, providers, endpoints, accounts, ledger entries, and settlement history.

**Mitigation:**
- Never expose DELETE to users (soft-delete or deactivate instead)
- Use `deleted_at` columns or status-based deactivation patterns
- Restrict DELETE to administrative database access only

**Severity:** High (data loss risk), but acceptably mitigated by not exposing DELETE in the application.

### 5.4 No compound index for the most critical query

**Problem:** The gateway's hot path query is:

```sql
SELECT b.balance FROM accounts a
JOIN api_keys k ON k.user_id = a.owner_id
WHERE k.key_hash = $1 AND k.status = 'active';
```

This touches `api_keys(key_hash)` then joins to `accounts(owner_id)`. There's no covering index.

**Mitigation:**
- `idx_api_keys_key_hash` covers the lookup
- `idx_accounts_owner` covers the join
- For MVP this is sufficient. If performance is an issue, a denormalized `account_id` on `api_keys` could eliminate the join entirely.

**Severity:** Low for MVP. Monitor in production.

### 5.5 `usage_events` will be the largest table by far

**Problem:** At even moderate traffic (100 req/s), `usage_events` grows by ~8.6M rows/day. Querying aggregates across this table without proper indexing will be slow.

**Mitigation:**
- Partitioning by `created_at` (monthly or daily ranges) should be planned from the start
- The settlement worker's aggregation query should use time-bounded scans
- Consider a materialized view for provider earnings if reports become slow

**Severity:** Medium. Not a problem at launch but will require attention as traffic grows.

### 5.6 No explicit unique constraint on `request_id`

**Problem:** The schema shows `request_id TEXT NOT NULL` but no explicit UNIQUE constraint. Idempotency enforcement depends on this being unique.

**Mitigation:** Add `UNIQUE (request_id)` to the `usage_events` table definition.

**Severity:** High. Without it, retry safety is broken and double-billing is possible.

### 5.7 `updated_at` is not automatically managed

**Problem:** `updated_at` columns on `users`, `providers`, `api_endpoints`, and `accounts` rely on application code to set them. If the application forgets, stale data silently accumulates.

**Mitigation:**
- Use a PostgreSQL trigger to auto-update `updated_at`:

```sql
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

**Severity:** Low. Cosmetic until stale data causes real bugs.

### 5.8 No tenant isolation

**Problem:** All data lives in a single schema. A bug in the provider dashboard could accidentally show one provider's endpoints to another. There's no built-in row-level security.

**Mitigation:**
- All queries must filter by `owner_id` (application-level enforcement)
- PostgreSQL Row-Level Security (RLS) can be added as a defense-in-depth layer
- For MVP, careful query construction is sufficient

**Severity:** Low for single-tenant MVP. Must be addressed before multi-tenant deployment.

### 5.9 Settlement currency mismatch

**Problem:** `settlement_entries.currency` defaults to XLM, but the actual distribution depends on the provider's configured payout address. If a provider sets a USDC-oriented Stellar wallet, sending XLM would fail on Stellar.

**Mitigation:**
- The settlement worker should read `users.payout_stellar_address` and match the currency to the expected asset
- For MVP with XLM-only, this is a non-issue until multi-currency support is added

**Severity:** Low for MVP. Must be addressed before adding USDC support.

---

## 6. Summary

| Strength | Concern |
|---|---|
| Internal credit and Stellar data are cleanly separated (no conflation) | `accounts.balance` can drift from ledger reality |
| `balance_after` enables fast audit and consistency checks | Polymorphic FK has no referential integrity |
| `request_id` provides idempotent metering | No UNIQUE constraint on `request_id` (must be added) |
| Denormalized `provider_id` avoids frequent joins | `usage_events` will grow fast — partition planning needed |
| Batched settlement model keeps gateway latency low | No automatic `updated_at` triggers |
| Stellar wallet references inlined on `users` (no separate table) | `ON DELETE CASCADE` risks in production |
