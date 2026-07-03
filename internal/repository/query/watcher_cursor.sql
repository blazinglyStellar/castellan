-- name: GetWatcherCursor :one
SELECT * FROM stellar_watcher_cursor
WHERE id = 1
LIMIT 1;

-- name: UpsertWatcherCursor :exec
INSERT INTO stellar_watcher_cursor (id, cursor, updated_at)
VALUES (1, $1, now())
ON CONFLICT (id)
DO UPDATE SET cursor = EXCLUDED.cursor, updated_at = now();
