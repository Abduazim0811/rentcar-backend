-- +goose Up

CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE rentals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    car_id BIGINT NOT NULL REFERENCES cars (id) ON DELETE RESTRICT,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    total_amount NUMERIC(12, 2) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending_payment',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT rentals_date_check CHECK (end_date >= start_date),
    CONSTRAINT rentals_total_amount_check CHECK (total_amount >= 0),
    CONSTRAINT rentals_status_check CHECK (status IN ('pending_payment', 'confirmed', 'cancelled', 'completed'))
);

CREATE INDEX idx_rentals_user_id ON rentals (user_id);
CREATE INDEX idx_rentals_car_id ON rentals (car_id);
CREATE INDEX idx_rentals_status ON rentals (status);
CREATE INDEX idx_rentals_car_dates ON rentals (car_id, start_date, end_date);
CREATE INDEX idx_rentals_availability
    ON rentals (car_id, start_date, end_date, status)
    WHERE status IN ('pending_payment', 'confirmed');

ALTER TABLE rentals
    ADD CONSTRAINT rentals_no_double_booking
    EXCLUDE USING gist (
        car_id WITH =,
        daterange(start_date, end_date, '[]') WITH &&
    )
    WHERE (status IN ('pending_payment', 'confirmed'));
