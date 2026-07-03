-- name: InsertDeposit :one
INSERT INTO deposits (
    account_id, from_address, amount, currency, memo, tx_hash, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetDepositByID :one
SELECT * FROM deposits
WHERE id = $1
LIMIT 1;

-- name: GetDepositByTxHash :one
SELECT * FROM deposits
WHERE tx_hash = $1
LIMIT 1;

-- name: ListDepositsByAccount :many
SELECT * FROM deposits
WHERE account_id = $1
ORDER BY created_at DESC;

-- name: UpdateDepositStatus :one
UPDATE deposits
SET status = $2
WHERE id = $1
RETURNING *;

-- name: ConfirmDeposit :one
UPDATE deposits
SET status = 'confirmed', confirmed_at = now()
WHERE id = $1
RETURNING *;

-- name: MarkDepositRejected :one
UPDATE deposits
SET status = 'failed', reason = $2
WHERE id = $1
RETURNING *;
