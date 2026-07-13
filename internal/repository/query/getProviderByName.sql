-- name: GetProviderByName :one
SELECT id, base_url, name, owner_id, status, description, created_at, updated_at
FROM providers
WHERE name = $1
  AND "status" = 'active'
LIMIT 1;
