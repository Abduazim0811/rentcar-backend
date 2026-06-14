-- +goose Up

WITH duplicate_phones AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY phone ORDER BY id) AS row_number
    FROM users
    WHERE phone <> ''
)
UPDATE users
SET phone = '',
    updated_at = NOW()
WHERE id IN (
    SELECT id
    FROM duplicate_phones
    WHERE row_number > 1
);

DROP INDEX IF EXISTS idx_users_phone;

CREATE UNIQUE INDEX idx_users_phone_unique
    ON users (phone)
    WHERE phone <> '';

-- +goose Down

DROP INDEX IF EXISTS idx_users_phone_unique;

CREATE INDEX idx_users_phone ON users (phone);
