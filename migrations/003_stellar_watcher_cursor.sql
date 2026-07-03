-- +goose Up

CREATE TABLE stellar_watcher_cursor (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    cursor      TEXT NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO stellar_watcher_cursor (id, cursor) VALUES (1, 'now');

ALTER TABLE deposits ADD COLUMN reason TEXT;

-- +goose Down

ALTER TABLE deposits DROP COLUMN IF EXISTS reason;
DROP TABLE IF EXISTS stellar_watcher_cursor;
