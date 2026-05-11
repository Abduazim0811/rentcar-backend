-- +goose Up

ALTER TABLE users
    ADD COLUMN email_verified_at TIMESTAMPTZ NULL,
    ADD COLUMN email_verification_code_hash TEXT NULL,
    ADD COLUMN email_verification_expires_at TIMESTAMPTZ NULL,
    ADD COLUMN email_verification_sent_at TIMESTAMPTZ NULL,
    ADD COLUMN email_verification_attempts INT NOT NULL DEFAULT 0;

UPDATE users
SET email_verified_at = NOW()
WHERE email_verified_at IS NULL;

CREATE INDEX idx_users_email_verification_expires_at
    ON users (email_verification_expires_at)
    WHERE email_verified_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_users_email_verification_expires_at;

ALTER TABLE users
    DROP COLUMN IF EXISTS email_verification_attempts,
    DROP COLUMN IF EXISTS email_verification_sent_at,
    DROP COLUMN IF EXISTS email_verification_expires_at,
    DROP COLUMN IF EXISTS email_verification_code_hash,
    DROP COLUMN IF EXISTS email_verified_at;
