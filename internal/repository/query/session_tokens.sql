-- name: InsertSessionToken :one
INSERT INTO session_tokens (user_id, token_hash, label, scope, status, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetSessionTokenByHash :one
SELECT id, user_id, token_hash, label, scope, status, expires_at, created_at
FROM session_tokens
WHERE token_hash = $1
LIMIT 1;

-- name: ListSessionTokensByUser :many
SELECT id, user_id, token_hash, label, scope, status, expires_at, created_at
FROM session_tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateSessionTokenStatus :one
UPDATE session_tokens
SET status = $2
WHERE id = $1
RETURNING *;
