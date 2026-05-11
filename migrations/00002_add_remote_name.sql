-- +goose Up
ALTER TABLE repo_configs ADD COLUMN remote_name VARCHAR(255) NOT NULL DEFAULT 'origin';

-- +goose Down
ALTER TABLE repo_configs DROP COLUMN remote_name;
