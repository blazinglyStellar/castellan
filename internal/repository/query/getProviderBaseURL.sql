-- name: GetProviderBaseURL :one
SELECT base_url
FROM providers 
WHERE id = $1 
AND "status" = 'active'
LIMIT 1;