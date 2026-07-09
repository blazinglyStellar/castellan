# End-to-End Local Test: Castellan with httpbin.org

Run the full gateway pipeline (AuthCheck → PricingResolver → BalanceCheck →
Reservation → Proxy → UsageCapture) against a real public API on localhost.

## Prerequisites

- Docker Compose (PostgreSQL 17 + Redis)
- A Stellar testnet wallet address (create at stellar laboratory)

## Step 0: create `.env`

```
cp .env.example .env
```

Set these values:

```
PORT=8080
STELLAR_HOT_WALLET_ADDRESS=<your-testnet-public-key>
SESSION_STORE_SECRET=<any-64-char-string>
DASHBOARD_URL=http://localhost:3000
```

## Step 1: start infra + migrate + seed

```bash
docker compose up -d postgres redis
make migrate
make seed
```

## Step 2: seed a real httpbin provider

Create `scripts/seed_httpbin_provider.sql`:

```sql
WITH ins_provider AS (
    INSERT INTO providers (owner_id, name, base_url, description, status)
    SELECT id, 'httpbin', 'https://httpbin.org', 'E2E test downstream', 'active'
    FROM users WHERE email = 'seed-provider@castellan.local'
    ON CONFLICT (name) DO NOTHING RETURNING id
)
INSERT INTO api_endpoints (provider_id, route, method, price_amount, currency, rate_limit, status, description)
SELECT (SELECT id FROM ins_provider), *
FROM (VALUES
    ('/post',       'POST',  '0.05', 30, 'Echo POST'),
    ('/get',        'GET',   '0.01', 60, 'Echo GET'),
    ('/status/200', 'GET',   '0.01', 60, '200 OK test'),
    ('/status/500', 'GET',   '0.01', 60, '500 failure test')
) AS t(route, method, price_amount, rate_limit, description)
WHERE EXISTS (SELECT 1 FROM ins_provider)
ON CONFLICT (provider_id, route, method) DO NOTHING;
```

```bash
psql "postgresql://postgres:postgres@localhost:5432/castellan?sslmode=disable" \
  -f scripts/seed_httpbin_provider.sql
```

## Step 3: create an API key

```bash
RAW_KEY="ca_$(openssl rand -base64 24 | tr -d '/+=')"
HASH=$(echo -n "$RAW_KEY" | sha256sum | cut -d' ' -f1)
echo "Raw key (save this): $RAW_KEY"
```

```sql
SELECT id FROM users WHERE email = 'amustee11@gmail.com';
INSERT INTO api_keys (user_id, key_hash, label, status, created_at)
VALUES ('<USER_UUID>', '<HASH>', 'e2e-test', 'active', now());
```

## Step 4: start the server

```bash
PORT=8080 go run cmd/api/main.go
```

## Step 5: curl through the gateway

```bash
PROV=$(psql -t -A "postgresql://postgres:postgres@localhost:5432/castellan?sslmode=disable" \
  -c "SELECT id FROM providers WHERE name = 'httpbin';")

# Happy path
curl -v -X POST "http://localhost:8080/api/gateway/${PROV}/post" \
  -H "Authorization: Bearer $RAW_KEY" \
  -H "Content-Type: application/json" \
  -d '{"hello":"castellan"}'

# GET with query params
curl -v "http://localhost:8080/api/gateway/${PROV}/get?foo=bar" \
  -H "Authorization: Bearer $RAW_KEY"

# Upstream 500
curl -v "http://localhost:8080/api/gateway/${PROV}/status/500" \
  -H "Authorization: Bearer $RAW_KEY"

# Invalid provider UUID
curl -v "http://localhost:8080/api/gateway/00000000-0000-0000-0000-000000000000/post" \
  -H "Authorization: Bearer $RAW_KEY"
```

## Middleware pipeline (in order)

AuthCheck → PricingResolver → RateLimitCheck → BalanceCheck → MaxBodySize →
Reservation → Proxy.director → [upstream] → UsageCapture

## Optional: skip the Stellar wallet

Change `server.go:70-72` from a hard error to a warning:

```go
if stellarCfg.HotWalletAddress == "" {
    slog.Warn("STELLAR_HOT_WALLET_ADDRESS not set -- deposits disabled")
}
```

The watcher no-ops with an empty address, gateway still works.

## Notes

- `PORT` has no code default — omitting it binds to a random port
- Makefile's `DB_PASSWORD` defaults to `1234`, `.env.example` uses `postgres`
- httpbin may rate-limit — switch to `jsonplaceholder.typicode.com` for GET-only
