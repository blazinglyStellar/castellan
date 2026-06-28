-- name: GetAccountBalance :one
SELECT balance
FROM accounts
WHERE owner_id = $1
LIMIT 1;