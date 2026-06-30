-- name: InsertLedgerEntry :one
INSERT INTO ledger_entries (
    account_id, entry_type, amount, balance_after, currency,
    reference_id, reference_type, status, description
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetLedgerEntryByID :one
SELECT * FROM ledger_entries
WHERE id = $1
LIMIT 1;

-- name: GetLedgerEntryByReferenceID :one
SELECT * FROM ledger_entries
WHERE reference_id = $1
LIMIT 1;

-- name: ListLedgerEntriesByAccount :many
SELECT * FROM ledger_entries
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListLedgerEntriesByAccountAndType :many
SELECT * FROM ledger_entries
WHERE account_id = $1 AND entry_type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateLedgerEntryStatus :one
UPDATE ledger_entries
SET status = $2
WHERE id = $1
RETURNING *;
