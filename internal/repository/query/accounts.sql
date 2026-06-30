-- name: GetOrCreateAccount :one
INSERT INTO accounts (owner_id, balance, currency)
VALUES ($1, 0, 'XLM')
ON CONFLICT (owner_id) DO UPDATE SET updated_at = now()
RETURNING *;

-- name: GetAccountByOwnerID :one
SELECT * FROM accounts
WHERE owner_id = $1
LIMIT 1;

-- name: UpdateAccountBalance :one
UPDATE accounts
SET balance = $2, updated_at = now()
WHERE id = $1
RETURNING *;
