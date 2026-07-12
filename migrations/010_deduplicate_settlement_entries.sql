-- +goose Up
-- Remove duplicate settlement_entries created by the old double-entry bug
-- (SubmitPayouts inserting entries that CreateBatch already created).
-- Keep only the latest entry per (batch_id, provider_id).

DELETE FROM settlement_entries
WHERE id IN (
    SELECT id FROM (
        SELECT id, ROW_NUMBER() OVER (
            PARTITION BY batch_id, provider_id
            ORDER BY created_at DESC
        ) AS rn
        FROM settlement_entries
    ) dup
    WHERE dup.rn > 1
);

-- +goose Down
-- No down migration: duplicates were a bug and cannot be recreated
-- by a single query.
