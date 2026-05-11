-- +goose Up

CREATE TABLE car_maintenances (
    id BIGSERIAL PRIMARY KEY,
    car_id BIGINT NOT NULL REFERENCES cars (id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    reason VARCHAR(255) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'scheduled',
    notes TEXT NOT NULL DEFAULT '',
    created_by BIGINT NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT car_maintenances_date_check CHECK (end_date >= start_date),
    CONSTRAINT car_maintenances_status_check CHECK (status IN ('scheduled', 'in_progress', 'completed', 'cancelled'))
);

CREATE INDEX idx_car_maintenances_car_id ON car_maintenances (car_id);
CREATE INDEX idx_car_maintenances_status ON car_maintenances (status);
CREATE INDEX idx_car_maintenances_car_dates ON car_maintenances (car_id, start_date, end_date);

ALTER TABLE car_maintenances
    ADD CONSTRAINT car_maintenances_no_overlap
    EXCLUDE USING gist (
        car_id WITH =,
        daterange(start_date, end_date, '[]') WITH &&
    )
    WHERE (status IN ('scheduled', 'in_progress'));
