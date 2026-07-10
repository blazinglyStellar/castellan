-- +goose Up
ALTER TABLE api_endpoints ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE api_endpoints DROP COLUMN IF EXISTS description;
