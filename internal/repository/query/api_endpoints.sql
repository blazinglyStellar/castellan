-- name: CreateEndpoint :one
INSERT INTO api_endpoints (provider_id, route, method, price_amount, currency, rate_limit, status, description)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetEndpointByID :one
SELECT *
FROM api_endpoints
WHERE id = $1
LIMIT 1;

-- name: ListEndpointsByProvider :many
SELECT *
FROM api_endpoints
WHERE provider_id = $1
  AND (sqlc.narg('status')::endpoint_status IS NULL OR status = sqlc.narg('status')::endpoint_status)
ORDER BY created_at DESC;

-- name: UpdateEndpoint :one
UPDATE api_endpoints
SET route = $2,
    method = $3,
    price_amount = $4,
    currency = $5,
    rate_limit = $6,
    description = $7,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateEndpointStatus :one
UPDATE api_endpoints
SET status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteEndpoint :one
DELETE FROM api_endpoints
WHERE id = $1
RETURNING *;

-- name: GetEndpointByProviderRouteMethod :one
SELECT *
FROM api_endpoints
WHERE provider_id = $1
  AND route = $2
  AND method = $3
LIMIT 1;
