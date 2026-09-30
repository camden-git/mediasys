-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version bigint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS token_version;
