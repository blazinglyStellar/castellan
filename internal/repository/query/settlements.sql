-- name: InsertSettlementBatch :one
INSERT INTO settlement_batches (
    status, total_amount, currency, entry_count, tx_hash
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetSettlementBatchByID :one
SELECT * FROM settlement_batches
WHERE id = $1
LIMIT 1;

-- name: GetUnsettledProviderEarnings :many
SELECT
    ue.provider_id,
    SUM(ue.request_cost)::numeric AS total_amount,
    ue.currency,
    u.payout_stellar_address
FROM usage_events ue
JOIN providers p ON p.id = ue.provider_id
JOIN users u ON u.id = p.owner_id
WHERE ue.status = 'completed'
  AND u.payout_stellar_address IS NOT NULL
  AND u.payout_stellar_address != ''
  AND ue.created_at > COALESCE((
      SELECT MAX(sb.completed_at)
      FROM settlement_entries se
      JOIN settlement_batches sb ON sb.id = se.batch_id
      WHERE se.provider_id = ue.provider_id
        AND sb.status = 'completed'
  ), 'epoch'::timestamptz)
GROUP BY ue.provider_id, ue.currency, u.payout_stellar_address;

-- name: CountSettlementBatches :one
SELECT COUNT(*) FROM settlement_batches;

-- name: ListSettlementBatches :many
SELECT * FROM settlement_batches
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListFailedSettlementBatches :many
SELECT * FROM settlement_batches
WHERE status = 'failed'
ORDER BY created_at DESC;

-- name: UpdateSettlementBatchStatus :one
UPDATE settlement_batches
SET
    status = $2,
    completed_at = CASE
        WHEN $2 IN ('completed'::batch_status, 'failed'::batch_status) THEN COALESCE(completed_at, now())
        WHEN $2 IN ('pending'::batch_status, 'processing'::batch_status) THEN NULL
        ELSE completed_at
    END
WHERE id = $1
RETURNING *;

-- name: InsertSettlementEntry :one
INSERT INTO settlement_entries (
    batch_id, provider_id, amount, currency, wallet_address, status
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetSettlementEntryByID :one
SELECT * FROM settlement_entries
WHERE id = $1
LIMIT 1;

-- name: ListSettlementEntriesByBatchWithProvider :many
SELECT se.*, p.name::text AS provider_name
FROM settlement_entries se
JOIN providers p ON p.id = se.provider_id
WHERE se.batch_id = $1
ORDER BY se.created_at DESC;

-- name: ListSettlementEntriesByBatchForOwner :many
SELECT DISTINCT ON (se.provider_id) se.*, p.name::text AS provider_name
FROM settlement_entries se
JOIN providers p ON p.id = se.provider_id
WHERE se.batch_id = $1
  AND p.owner_id = $2
ORDER BY se.provider_id, se.created_at DESC;

-- name: ListSettlementEntriesByBatch :many
SELECT * FROM settlement_entries
WHERE batch_id = $1
ORDER BY created_at DESC;

-- name: GetSettlementEntryByBatchAndProvider :one
SELECT * FROM settlement_entries
WHERE batch_id = $1 AND provider_id = $2
LIMIT 1;

-- name: UpdateSettlementEntryStatus :one
UPDATE settlement_entries
SET status = $2
WHERE id = $1
RETURNING *;
