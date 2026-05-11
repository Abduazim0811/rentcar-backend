-- +goose Up

UPDATE cars
SET status = 'available',
    updated_at = NOW()
WHERE status = 'rented';
