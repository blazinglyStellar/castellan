-- +goose Up
UPDATE users SET deposit_memo = NULL;
ALTER TABLE users ALTER COLUMN deposit_memo TYPE VARCHAR(26);

-- +goose Down
ALTER TABLE users ALTER COLUMN deposit_memo TYPE TEXT;
