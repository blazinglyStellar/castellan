-- name: ListSettlementBatchesByOwnerFiltered :many
SELECT DISTINCT sb.*
FROM settlement_batches sb
JOIN settlement_entries se ON se.batch_id = sb.id
WHERE se.provider_id IN (SELECT id FROM providers WHERE owner_id = $1)
  AND ($5::text = '' OR sb.status = $5::batch_status)
  AND ($3::timestamptz = '0001-01-01'::timestamptz OR (sb.created_at, sb.id) < ($3::timestamptz, $4::uuid))
ORDER BY sb.created_at DESC, sb.id DESC
LIMIT $2;

-- name: ListSettlementBatchesByOwner :many
SELECT DISTINCT sb.*
FROM settlement_batches sb
JOIN settlement_entries se ON se.batch_id = sb.id
WHERE se.provider_id IN (SELECT id FROM providers WHERE owner_id = $1)
  AND ($3::timestamptz = '0001-01-01'::timestamptz OR (sb.created_at, sb.id) < ($3::timestamptz, $4::uuid))
ORDER BY sb.created_at DESC, sb.id DESC
LIMIT $2;

-- name: GetSettlementSummaryByOwner :one
SELECT COALESCE(SUM(se.amount), 0)::numeric AS total_settled
FROM settlement_entries se
JOIN settlement_batches sb ON sb.id = se.batch_id
WHERE se.provider_id IN (SELECT id FROM providers WHERE owner_id = $1)
  AND sb.status = 'completed';

-- name: GetSettlementMonthlyHistoryByOwner :many
SELECT
  DATE_TRUNC('month', sb.completed_at)::timestamptz AS month,
  SUM(se.amount)::numeric AS amount
FROM settlement_entries se
JOIN settlement_batches sb ON sb.id = se.batch_id
WHERE se.provider_id IN (SELECT id FROM providers WHERE owner_id = $1)
  AND sb.status = 'completed'
  AND sb.completed_at IS NOT NULL
GROUP BY DATE_TRUNC('month', sb.completed_at)
ORDER BY month DESC
LIMIT $2;
