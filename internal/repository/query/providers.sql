-- name: CreateProvider :one
INSERT INTO providers (owner_id, name, base_url, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetProviderByID :one
SELECT id, owner_id, name, base_url, status, created_at, updated_at
FROM providers
WHERE id = $1
LIMIT 1;

-- name: ListProvidersByOwner :many
SELECT id, owner_id, name, base_url, status, created_at, updated_at
FROM providers
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: ListAllProviders :many
SELECT id, owner_id, name, base_url, status, created_at, updated_at
FROM providers
WHERE (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC;

-- name: UpdateProvider :one
UPDATE providers
SET name = $2,
    base_url = $3,
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
