-- +goose Up

CREATE TABLE invoices (
    id BIGSERIAL PRIMARY KEY,
    rental_id BIGINT NOT NULL UNIQUE REFERENCES rentals (id) ON DELETE RESTRICT,
    invoice_number VARCHAR(40) NOT NULL UNIQUE,
    amount NUMERIC(12, 2) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'issued',
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    due_at TIMESTAMPTZ NULL,
    paid_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT invoices_amount_check CHECK (amount >= 0),
    CONSTRAINT invoices_status_check CHECK (status IN ('issued', 'paid', 'void'))
);

CREATE INDEX idx_invoices_rental_id ON invoices (rental_id);
CREATE INDEX idx_invoices_status ON invoices (status);
