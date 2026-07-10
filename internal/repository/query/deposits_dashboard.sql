-- name: ListDepositsByAccountCursor :many
SELECT * FROM deposits
WHERE account_id = $1
  AND ($3::timestamptz = '0001-01-01'::timestamptz OR (created_at, id) < ($3::timestamptz, $4::uuid))
ORDER BY created_at DESC, id DESC
LIMIT $2;
