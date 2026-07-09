-- +goose Up
ALTER TABLE providers ADD CONSTRAINT providers_name_key UNIQUE (name);

-- +goose Down
ALTER TABLE providers DROP CONSTRAINT IF EXISTS providers_name_key;
