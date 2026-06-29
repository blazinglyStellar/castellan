-- +goose Up

CREATE TYPE session_token_status AS ENUM ('active', 'revoked', 'expired');

CREATE TABLE session_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    label       TEXT,
    scope       TEXT,
    status      session_token_status NOT NULL DEFAULT 'active',
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_session_tokens_user ON session_tokens (user_id);
CREATE INDEX idx_session_tokens_status ON session_tokens (status);

-- +goose Down

DROP INDEX IF EXISTS idx_session_tokens_status;
DROP INDEX IF EXISTS idx_session_tokens_user;
DROP TABLE IF EXISTS session_tokens;
DROP TYPE IF EXISTS session_token_status;
