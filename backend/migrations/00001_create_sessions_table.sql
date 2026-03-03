-- +goose Up
CREATE TABLE IF NOT EXISTS sessions (
    session_uuid TEXT PRIMARY KEY,
    nickname TEXT NOT NULL,
    telegram_id INTEGER NOT NULL DEFAULT 0,
    telegram_username TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_sessions_telegram_id ON sessions(telegram_id) WHERE telegram_id != 0;

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_telegram_id;
DROP TABLE IF EXISTS sessions;
