-- +goose Up

CREATE TABLE cars (
    id BIGSERIAL PRIMARY KEY,
    brand VARCHAR(80) NOT NULL,
    model VARCHAR(80) NOT NULL,
    year INT NOT NULL,
    plate_number VARCHAR(30) NOT NULL UNIQUE,
    daily_rate NUMERIC(12, 2) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT cars_year_check CHECK (year >= 1900),
    CONSTRAINT cars_daily_rate_check CHECK (daily_rate > 0),
    CONSTRAINT cars_status_check CHECK (status IN ('available', 'rented', 'maintenance', 'inactive'))
);

CREATE INDEX idx_cars_status ON cars (status);
CREATE INDEX idx_cars_brand_model ON cars (brand, model);
