-- name: CreateProvider :one
INSERT INTO providers (owner_id, name, base_url, description, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetProviderByID :one
SELECT *
FROM providers
WHERE id = $1
LIMIT 1;

-- name: ListProvidersByOwner :many
SELECT *
FROM providers
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: ListAllProviders :many
SELECT *
FROM providers
WHERE (sqlc.narg('status')::provider_status IS NULL OR status = sqlc.narg('status')::provider_status)
ORDER BY created_at DESC;

-- name: UpdateProvider :one
UPDATE providers
SET name = $2,
    base_url = $3,
    description = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateProviderStatus :one
UPDATE providers
SET status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteProvider :one
DELETE FROM providers
WHERE id = $1
RETURNING *;

-- name: GetProviderStats :many
SELECT
  p.id,
  COALESCE(COUNT(DISTINCT ae.id), 0)::bigint AS endpoint_count,
  COALESCE(COUNT(DISTINCT ue.id), 0)::bigint AS total_calls,
  COALESCE(COUNT(DISTINCT ue.consumer_id), 0)::bigint AS active_consumers
FROM providers p
LEFT JOIN api_endpoints ae ON ae.provider_id = p.id
LEFT JOIN usage_events ue ON ue.endpoint_id = ae.id
GROUP BY p.id;
