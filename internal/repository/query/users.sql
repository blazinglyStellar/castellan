-- name: EnsureUserDepositMemo :one
UPDATE users
SET deposit_memo = COALESCE(deposit_memo, sqlc.arg('new_memo')::VARCHAR(26))
WHERE id = $1
RETURNING deposit_memo;

-- name: GetUserByDepositMemo :one
SELECT * FROM users
WHERE deposit_memo = $1
LIMIT 1;

-- name: UpsertUserByEmail :one
INSERT INTO users (email)
VALUES ($1)
ON CONFLICT (email) DO UPDATE
  SET updated_at = NOW()
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;
