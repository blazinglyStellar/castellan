-- +goose Up
ALTER TABLE users
  ADD COLUMN onboarding_completed BOOLEAN NOT NULL DEFAULT false;

UPDATE users SET onboarding_completed = true;

-- +goose Down
ALTER TABLE users DROP COLUMN onboarding_completed;
