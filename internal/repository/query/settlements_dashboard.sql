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
SELECT COALESCE(SUM(sub.amount), 0)::numeric AS total_settled
FROM (
    SELECT DISTINCT ON (se.batch_id, se.provider_id) se.amount
    FROM settlement_entries se
    JOIN settlement_batches sb ON sb.id = se.batch_id
    WHERE se.provider_id IN (SELECT id FROM providers WHERE owner_id = $1)
      AND sb.status = 'completed'
    ORDER BY se.batch_id, se.provider_id, se.created_at DESC
) sub;

-- name: GetSettlementSummaryByOwnerInRange :one
SELECT COALESCE(SUM(sub.amount), 0)::numeric AS total_settled
FROM (
    SELECT DISTINCT ON (se.batch_id, se.provider_id) se.amount
    FROM settlement_entries se
    JOIN settlement_batches sb ON sb.id = se.batch_id
    WHERE se.provider_id IN (SELECT id FROM providers WHERE owner_id = $1)
      AND sb.status = 'completed'
      AND sb.created_at >= $2::timestamptz
      AND sb.created_at <= $3::timestamptz
    ORDER BY se.batch_id, se.provider_id, se.created_at DESC
) sub;

-- name: GetSettlementMonthlyHistoryByOwner :many
SELECT
  DATE_TRUNC('month', sub.completed_at)::timestamptz AS month,
  SUM(sub.amount)::numeric AS amount
FROM (
    SELECT DISTINCT ON (se.batch_id, se.provider_id) se.amount, sb.completed_at
    FROM settlement_entries se
    JOIN settlement_batches sb ON sb.id = se.batch_id
    WHERE se.provider_id IN (SELECT id FROM providers WHERE owner_id = $1)
      AND sb.status = 'completed'
      AND sb.completed_at IS NOT NULL
    ORDER BY se.batch_id, se.provider_id, se.created_at DESC
) sub
GROUP BY DATE_TRUNC('month', sub.completed_at)
ORDER BY month DESC
LIMIT $2;
