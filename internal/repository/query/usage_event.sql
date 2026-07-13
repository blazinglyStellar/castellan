-- name: CreateUsageEvent :one
WITH inserted AS (
    INSERT INTO usage_events (
        consumer_id,
        provider_id,
        endpoint_id,
        request_cost,
        currency,
        status_code,
        latency_ms,
        response_size,
        request_id,
        status
    ) VALUES (
        $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
    )
    ON CONFLICT (request_id) DO NOTHING
    RETURNING *
)
SELECT * FROM inserted
UNION ALL
SELECT * FROM usage_events
WHERE request_id = $9
  AND NOT EXISTS (SELECT 1 FROM inserted);

-- name: GetUsageEventByRequestID :one
SELECT * FROM usage_events
WHERE request_id = $1;

-- name: ListUsageEventsByConsumer :many
SELECT * FROM usage_events
WHERE consumer_id = $1
ORDER BY created_at DESC;
