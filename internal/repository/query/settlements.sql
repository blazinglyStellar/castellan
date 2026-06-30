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

-- name: ListSettlementBatches :many
SELECT * FROM settlement_batches
ORDER BY created_at DESC;

-- name: UpdateSettlementBatchStatus :one
UPDATE settlement_batches
SET status = $2
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

-- name: ListSettlementEntriesByBatch :many
SELECT * FROM settlement_entries
WHERE batch_id = $1
ORDER BY created_at DESC;

-- name: UpdateSettlementEntryStatus :one
UPDATE settlement_entries
SET status = $2
WHERE id = $1
RETURNING *;
