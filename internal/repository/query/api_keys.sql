-- name: GetKeyByHash :one
SELECT id, user_id, key_hash, label, status, created_at, expires_at
FROM api_keys
WHERE key_hash = $1
LIMIT 1;

-- name: InsertKey :one
INSERT INTO api_keys (user_id, key_hash, label, status, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListKeysByUser :many
SELECT id, user_id, key_hash, label, status, created_at, expires_at
FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateKeyStatus :one
UPDATE api_keys
SET status = $2
WHERE id = $1
RETURNING *;

-- name: DeleteKey :one
DELETE FROM api_keys
WHERE id = $1
RETURNING *;

-- name: RevokeKey :one
UPDATE api_keys
SET status = 'revoked'
WHERE id = $1 AND status != 'revoked'
RETURNING *;

-- name: GetKeyByID :one
SELECT id, user_id, key_hash, label, status, created_at, expires_at
FROM api_keys
WHERE id = $1
LIMIT 1;

-- name: GetKeyWithUserAndAccount :one
SELECT
    k.id AS api_key_id,
    k.user_id,
    k.key_hash,
    k.label,
    k.status,
    k.created_at,
    k.expires_at,
    u.email,
    a.id AS account_id,
    a.balance,
    a.currency
FROM api_keys k
JOIN users u ON k.user_id = u.id
JOIN accounts a ON u.id = a.owner_id
WHERE k.key_hash = $1
LIMIT 1;
