-- +goose Up

ALTER TABLE cars ADD COLUMN IF NOT EXISTS image_url TEXT;
