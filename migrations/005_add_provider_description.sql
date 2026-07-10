-- +goose Up
ALTER TABLE providers ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE providers DROP COLUMN IF EXISTS description;
