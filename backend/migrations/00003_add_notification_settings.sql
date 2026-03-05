-- +goose Up
ALTER TABLE sessions ADD COLUMN notify_threshold INTEGER NOT NULL DEFAULT 2;
ALTER TABLE sessions ADD COLUMN last_notified_at INTEGER;

-- +goose Down
ALTER TABLE sessions DROP COLUMN last_notified_at;
ALTER TABLE sessions DROP COLUMN notify_threshold;
