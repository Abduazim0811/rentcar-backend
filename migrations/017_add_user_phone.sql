-- +goose Up

ALTER TABLE users
    ADD COLUMN phone VARCHAR(32) NOT NULL DEFAULT '';

CREATE INDEX idx_users_phone ON users (phone);

-- +goose Down

DROP INDEX IF EXISTS idx_users_phone;

ALTER TABLE users
    DROP COLUMN IF EXISTS phone;
