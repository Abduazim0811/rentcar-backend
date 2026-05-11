package repository

import (
	"context"
	"database/sql"

	"car-rental-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceRepository interface {
	Create(ctx context.Context, invoice *models.Invoice) error
	FindByRentalID(ctx context.Context, rentalID int64) (*models.Invoice, error)
	MarkPaid(ctx context.Context, rentalID int64) error
}

type InvoicePostgresRepository struct {
	db *pgxpool.Pool
}

func NewInvoicePostgresRepository(db *pgxpool.Pool) *InvoicePostgresRepository {
	return &InvoicePostgresRepository{db: db}
}

func (r *InvoicePostgresRepository) Create(ctx context.Context, invoice *models.Invoice) error {
	if invoice.Status == "" {
		invoice.Status = models.InvoiceStatusIssued
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO invoices (rental_id, invoice_number, amount, status, issued_at, due_at, paid_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, issued_at, created_at, updated_at
	`, invoice.RentalID, invoice.InvoiceNumber, invoice.Amount, invoice.Status, invoice.IssuedAt, invoice.DueAt, invoice.PaidAt).
		Scan(&invoice.ID, &invoice.IssuedAt, &invoice.CreatedAt, &invoice.UpdatedAt)
	return mapPostgresError(err)
}

func (r *InvoicePostgresRepository) FindByRentalID(ctx context.Context, rentalID int64) (*models.Invoice, error) {
	invoice, err := scanInvoice(r.db.QueryRow(ctx, `
		SELECT id, rental_id, invoice_number, amount, status, issued_at, due_at, paid_at, created_at, updated_at
		FROM invoices
		WHERE rental_id = $1
	`, rentalID))
	if err != nil {
		return nil, mapPostgresError(err)
	}
	return invoice, nil
}

func (r *InvoicePostgresRepository) MarkPaid(ctx context.Context, rentalID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE invoices
		SET status = 'paid', paid_at = NOW(), updated_at = NOW()
		WHERE rental_id = $1 AND status = 'issued'
	`, rentalID)
	return mapPostgresError(err)
}

func scanInvoice(row pgx.Row) (*models.Invoice, error) {
	var invoice models.Invoice
	var dueAt sql.NullTime
	var paidAt sql.NullTime
	if err := row.Scan(&invoice.ID, &invoice.RentalID, &invoice.InvoiceNumber, &invoice.Amount, &invoice.Status, &invoice.IssuedAt, &dueAt, &paidAt, &invoice.CreatedAt, &invoice.UpdatedAt); err != nil {
		return nil, err
	}
	if dueAt.Valid {
		invoice.DueAt = &dueAt.Time
	}
	if paidAt.Valid {
		invoice.PaidAt = &paidAt.Time
	}
	return &invoice, nil
}
