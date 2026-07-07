-- name: ListUsageEventsByConsumerCursor :many
SELECT ue.*, ep.route, ep.method
FROM usage_events ue
JOIN api_endpoints ep ON ep.id = ue.endpoint_id
WHERE ue.consumer_id = $1
  AND ($3::timestamptz = '0001-01-01'::timestamptz OR (ue.created_at, ue.id) < ($3::timestamptz, $4::uuid))
ORDER BY ue.created_at DESC, ue.id DESC
LIMIT $2;

-- name: ListUsageEventsByConsumerFiltered :many
SELECT ue.*, ep.route, ep.method
FROM usage_events ue
JOIN api_endpoints ep ON ep.id = ue.endpoint_id
WHERE ue.consumer_id = $1
  AND ($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR ue.endpoint_id = $2)
  AND ($3 = 0 OR ue.status_code = $3)
  AND ($4::timestamptz = '0001-01-01'::timestamptz OR ue.created_at >= $4)
  AND ($5::timestamptz = '0001-01-01'::timestamptz OR ue.created_at <= $5)
  AND ($7::timestamptz = '0001-01-01'::timestamptz OR (ue.created_at, ue.id) < ($7::timestamptz, $8::uuid))
ORDER BY ue.created_at DESC, ue.id DESC
LIMIT $6;

-- name: ListUsageEventsByProviderCursor :many
SELECT ue.*, ep.route, ep.method
FROM usage_events ue
JOIN api_endpoints ep ON ep.id = ue.endpoint_id
WHERE ue.provider_id = $1
  AND ($3::timestamptz = '0001-01-01'::timestamptz OR (ue.created_at, ue.id) < ($3::timestamptz, $4::uuid))
ORDER BY ue.created_at DESC, ue.id DESC
LIMIT $2;

-- name: ListUsageEventsByProviderFiltered :many
SELECT ue.*, ep.route, ep.method
FROM usage_events ue
JOIN api_endpoints ep ON ep.id = ue.endpoint_id
WHERE ue.provider_id = $1
  AND ($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR ue.endpoint_id = $2)
  AND ($3 = 0 OR ue.status_code = $3)
  AND ($4::timestamptz = '0001-01-01'::timestamptz OR ue.created_at >= $4)
  AND ($5::timestamptz = '0001-01-01'::timestamptz OR ue.created_at <= $5)
  AND ($7::timestamptz = '0001-01-01'::timestamptz OR (ue.created_at, ue.id) < ($7::timestamptz, $8::uuid))
ORDER BY ue.created_at DESC, ue.id DESC
LIMIT $6;

-- name: GetTotalEarningsByProvider :one
SELECT COALESCE(SUM(request_cost), 0)::numeric AS total
FROM usage_events
WHERE provider_id = $1 AND status = 'completed';

-- name: GetEarningsByEndpoint :many
SELECT ue.endpoint_id, ep.route, SUM(ue.request_cost)::numeric AS total
FROM usage_events ue
JOIN api_endpoints ep ON ep.id = ue.endpoint_id
WHERE ue.provider_id = $1 AND ue.status = 'completed'
GROUP BY ue.endpoint_id, ep.route;

-- name: GetEarningsSparkline :many
SELECT DATE(created_at) AS date, SUM(request_cost)::numeric AS amount
FROM usage_events
WHERE provider_id = $1 AND status = 'completed'
  AND created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY date;

-- name: GetUnsettledEarningsByProvider :one
SELECT COALESCE(SUM(le.amount), 0)::numeric AS total
FROM ledger_entries le
WHERE le.account_id = (SELECT a.id FROM accounts a JOIN providers p ON p.owner_id = a.owner_id WHERE p.id = $1)
  AND le.entry_type = 'settlement'
  AND le.status = 'pending';

-- name: GetActiveReservationsSum :one
SELECT COALESCE(SUM(amount), 0)::numeric AS total
FROM ledger_entries
WHERE account_id = $1
  AND entry_type = 'reservation'
  AND status = 'pending';

-- name: GetUserProviderCount :one
SELECT COUNT(*) AS count
FROM providers
WHERE owner_id = $1;
