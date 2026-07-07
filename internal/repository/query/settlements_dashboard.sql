-- name: ListSettlementBatchesByProvider :many
SELECT DISTINCT sb.*
FROM settlement_batches sb
JOIN settlement_entries se ON se.batch_id = sb.id
WHERE se.provider_id = $1
  AND ($3::timestamptz = '0001-01-01'::timestamptz OR (sb.created_at, sb.id) < ($3::timestamptz, $4::uuid))
ORDER BY sb.created_at DESC, sb.id DESC
LIMIT $2;
