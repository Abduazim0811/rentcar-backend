-- +goose Up

CREATE TABLE payments (
    id BIGSERIAL PRIMARY KEY,
    rental_id BIGINT NOT NULL UNIQUE REFERENCES rentals (id) ON DELETE RESTRICT,
    amount NUMERIC(12, 2) NOT NULL,
    method VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    paid_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT payments_amount_check CHECK (amount >= 0),
    CONSTRAINT payments_method_check CHECK (method IN ('cash', 'card', 'bank_transfer')),
    CONSTRAINT payments_status_check CHECK (status IN ('pending', 'paid', 'failed', 'refunded'))
);

CREATE INDEX idx_payments_rental_id ON payments (rental_id);
CREATE INDEX idx_payments_status ON payments (status);
