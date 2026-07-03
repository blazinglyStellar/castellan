-- name: EnsureUserDepositMemo :one
UPDATE users
SET deposit_memo = COALESCE(deposit_memo, gen_random_uuid()::text)
WHERE id = $1
RETURNING deposit_memo;

-- name: GetUserByDepositMemo :one
SELECT * FROM users
WHERE deposit_memo = $1
LIMIT 1;
