-- +goose Up
ALTER TABLE sessions ADD COLUMN last_login_at INTEGER;

-- +goose Down
-- SQLite 3.35+ supports DROP COLUMN
ALTER TABLE sessions DROP COLUMN last_login_at;
