# End-to-End Local Test Plan: Castellan with a Real Public API

## Goal

Hit Castellan's gateway with curl, have it forward to a real public API
(https://httpbin.org), deduct from a prepaid balance, and return the
upstream response -- all on localhost with no cloud provisioning.

## Chosen Downstream API

[/post] POST  JSONEcho
[/get]  GET   Query echo
[/status/XXX] GET  Status code echo
Ideal for proxy testing.

## Prerequisites

| Item | Why | How |
|---|---|---|
| Docker Compose | PostgreSQL 17 + Redis | docker compose up -d postgres redis |
| Stellar testnet wallet | Required by server.go L70-72 | Create at https://laboratory.stellar.org/#account-creator?network=testnet |
| .env file | App reads env vars | Copy .env.example, fill STELLAR_HOT_WALLET_ADDRESS + SESSION_STORE_SECRET + DASHBOARD_URL |

Stellar address is not used in the request path -- only deposits/settlement.
The gateway works without a functioning watcher.

## Steps

### 1. Infrastructure

```bash
docker compose up -d postgres redis
```

### 2. Environment

```bash
cp .env.example .env
# Edit: STELLAR_HOT_WALLET_ADDRESS=<testnet-key>
# Edit: SESSION_STORE_SECRET=<64-char-string>
# Edit: DASHBOARD_URL=http://localhost:3000
# Edit: PORT=8080
```

### 3. Migrate + seed

```bash
make migrate
make seed
```

Creates:
- Users: amustee11@gmail.com (consumer), seed-provider@castellan.local
- Providers with fake base URLs (not usable as-is)
- Consumer account with 1000 XLM

### 4. Seed a real httpbin provider

File scripts/seed_httpbin_provider.sql:

```sql
WITH ins_provider AS (
    INSERT INTO providers (owner_id, name, base_url, description, status)
    SELECT id, 'httpbin', 'https://httpbin.org', 'E2E test downstream', 'active'
    FROM users WHERE email = 'seed-provider@castellan.local'
    ON CONFLICT (name) DO NOTHING
    RETURNING id
)
INSERT INTO api_endpoints (provider_id, route, method, price_amount, currency, rate_limit, status, description)
SELECT
    (SELECT id FROM ins_provider),
    route, method, price_amount, 'XLM', rate_limit, 'active', description
FROM (VALUES
    ('/post',       'POST',  '0.05', 30, 'Echo POST'),
    ('/get',        'GET',   '0.01', 60, 'Echo GET'),
    ('/status/200', 'GET',   '0.01', 60, '200 OK test'),
    ('/status/500', 'GET',   '0.01', 60, '500 failure test')
) AS t(route, method, price_amount, rate_limit, description)
WHERE EXISTS (SELECT 1 FROM ins_provider)
ON CONFLICT (provider_id, route, method) DO NOTHING;
```

Run:
```bash
psql "postgresql://postgres:postgres@localhost:5432/castellan?sslmode=disable" -f scripts/seed_httpbin_provider.sql
```

### 5. Create an API key (direct SQL -- no OAuth needed)

```bash
RAW_KEY="ca_$(openssl rand -base64 32 | tr -d '/+=' | cut -c1-43)"
HASH=$(echo -n "$RAW_KEY" | sha256sum | cut -d' ' -f1)
echo "Key: $RAW_KEY"
echo "Hash: $HASH"
```

```sql
SELECT id FROM users WHERE email = 'amustee11@gmail.com';
INSERT INTO api_keys (user_id, key_hash, label, status, created_at)
VALUES ('<USER_UUID>', '<KEY_HASH>', 'e2e-test-key', 'active', now());
```

### 6. Start Castellan

```bash
PORT=8080 go run cmd/api/main.go
```

### 7. Test the full flow

```bash
# Get the httpbin provider UUID
psql "...castellan?sslmode=disable" -c "SELECT id FROM providers WHERE name = 'httpbin';"

# Happy path -- POST through the gateway
curl -v -X POST "http://localhost:8080/api/gateway/<PROVIDER_UUID>/post" \
  -H "Authorization: Bearer $RAW_KEY" \
  -H "Content-Type: application/json" \
  -d '{"hello":"castellan"}'

# GET with query params
curl -v "http://localhost:8080/api/gateway/<PROVIDER_UUID>/get?foo=bar" \
  -H "Authorization: Bearer $RAW_KEY"

# Upstream 500 (should still bill? check behavior)
curl -v "http://localhost:8080/api/gateway/<PROVIDER_UUID>/status/500" \
  -H "Authorization: Bearer $RAW_KEY"

# Invalid provider UUID
curl -v "http://localhost:8080/api/gateway/00000000-0000-0000-0000-000000000000/post" \
  -H "Authorization: Bearer $RAW_KEY"
```

Expected behavior for happy path:
1. AuthCheck -- validates bearer token, sets ConsumerInfo
2. PricingResolver -- parses /api/gateway/{uuid}/post, looks up endpoint, sets price 0.05 XLM
3. RateLimitCheck -- passes (rate limit not hit)
4. BalanceCheck -- 1000 >= 0.05
5. MaxBodySize -- passes
6. Reservation -- reserves 0.05
7. Proxy.director -- parses provider UUID, resolves base_url to https://httpbin.org, rewrites to /post
8. Proxy forwards to https://httpbin.org/post
9. UsageCapture -- records usage, commits reservation, deducts 0.05 XLM
10. httpbin response returned to client

## Risks

| Risk | Mitigation |
|---|---|
| PORT=0 if unset | Set PORT=8080 in .env |
| httpbin rate-limits | Switch to jsonplaceholder.typicode.com or local python -m http.server |
| No API keys in seed | Direct SQL insert is one line |
| OAuth not wired | API keys work without OAuth |
