-- +goose Up
ALTER TABLE sessions ADD COLUMN launcher_version TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sessions DROP COLUMN launcher_version;

