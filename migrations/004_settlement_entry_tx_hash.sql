-- +goose Up
ALTER TABLE settlement_entries ADD COLUMN tx_hash TEXT;

-- +goose Down
ALTER TABLE settlement_entries DROP COLUMN tx_hash;
