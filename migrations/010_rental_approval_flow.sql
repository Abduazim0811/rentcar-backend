-- +goose Up

ALTER TABLE rentals DROP CONSTRAINT IF EXISTS rentals_no_double_booking;
DROP INDEX IF EXISTS idx_rentals_availability;

ALTER TABLE rentals DROP CONSTRAINT IF EXISTS rentals_status_check;
ALTER TABLE rentals
    ADD CONSTRAINT rentals_status_check
    CHECK (status IN ('requested', 'approved', 'rejected', 'pending_payment', 'confirmed', 'active', 'cancelled', 'completed'));

CREATE INDEX idx_rentals_availability
    ON rentals (car_id, start_date, end_date, status)
    WHERE status IN ('requested', 'approved', 'pending_payment', 'confirmed', 'active');

ALTER TABLE rentals
    ADD CONSTRAINT rentals_no_double_booking
    EXCLUDE USING gist (
        car_id WITH =,
        daterange(start_date, end_date, '[]') WITH &&
    )
    WHERE (status IN ('requested', 'approved', 'pending_payment', 'confirmed', 'active'));
