package repository

import (
	"context"
	"errors"

	"car-rental-system/internal/models"
	"car-rental-system/pkg/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	CreateForRental(ctx context.Context, payment *models.Payment, expectedRentalStatus models.RentalStatus) error
	FindByID(ctx context.Context, id int64) (*models.Payment, error)
	FindByRentalID(ctx context.Context, rentalID int64) (*models.Payment, error)
	UpdateStatus(ctx context.Context, id int64, status models.PaymentStatus) error
	Confirm(ctx context.Context, id int64) error
	Fail(ctx context.Context, id int64) error
	Refund(ctx context.Context, id int64) error
}

type PaymentPostgresRepository struct {
	db *pgxpool.Pool
}

func NewPaymentPostgresRepository(db *pgxpool.Pool) *PaymentPostgresRepository {
	return &PaymentPostgresRepository{db: db}
}

func (r *PaymentPostgresRepository) Create(ctx context.Context, payment *models.Payment) error {
	query := `
		INSERT INTO payments (rental_id, amount, method, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	if payment.Status == "" {
		payment.Status = models.PaymentStatusPending
	}

	err := r.db.QueryRow(ctx, query, payment.RentalID, payment.Amount, payment.Method, payment.Status).
		Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt)
	if isPostgresConstraint(err, "23505", "payments_rental_id_key") {
		return apperror.ErrPaymentExists
	}
	return mapPostgresError(err)
}

func (r *PaymentPostgresRepository) CreateForRental(ctx context.Context, payment *models.Payment, expectedRentalStatus models.RentalStatus) error {
	if payment.Status == "" {
		payment.Status = models.PaymentStatusPending
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	var currentStatus models.RentalStatus
	err = tx.QueryRow(ctx, `SELECT status FROM rentals WHERE id = $1 FOR UPDATE`, payment.RentalID).Scan(&currentStatus)
	if err != nil {
		return mapPostgresError(err)
	}
	if currentStatus != expectedRentalStatus {
		return apperror.New(409, "rental status changed; refresh and try again")
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO payments (rental_id, amount, method, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, payment.RentalID, payment.Amount, payment.Method, payment.Status).
		Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt)
	if isPostgresConstraint(err, "23505", "payments_rental_id_key") {
		return apperror.ErrPaymentExists
	}
	if err != nil {
		return mapPostgresError(err)
	}

	if currentStatus == models.RentalStatusApproved {
		result, err := tx.Exec(ctx, `
			UPDATE rentals
			SET status = $1, updated_at = NOW()
			WHERE id = $2
			  AND status = 'approved'
		`, models.RentalStatusPendingPayment, payment.RentalID)
		if err != nil {
			return mapPostgresError(err)
		}
		if result.RowsAffected() == 0 {
			return apperror.New(409, "rental status changed; refresh and try again")
		}
	}

	return mapPostgresError(tx.Commit(ctx))
}

func (r *PaymentPostgresRepository) FindByID(ctx context.Context, id int64) (*models.Payment, error) {
	query := `
		SELECT id, rental_id, amount, method, status, paid_at, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	payment, err := scanPayment(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, mapPostgresError(err)
	}

	return payment, nil
}

func (r *PaymentPostgresRepository) FindByRentalID(ctx context.Context, rentalID int64) (*models.Payment, error) {
	query := `
		SELECT id, rental_id, amount, method, status, paid_at, created_at, updated_at
		FROM payments
		WHERE rental_id = $1
	`

	payment, err := scanPayment(r.db.QueryRow(ctx, query, rentalID))
	if err != nil {
		return nil, mapPostgresError(err)
	}

	return payment, nil
}

func (r *PaymentPostgresRepository) UpdateStatus(ctx context.Context, id int64, status models.PaymentStatus) error {
	query := `
		UPDATE payments
		SET status = $1,
		    paid_at = CASE WHEN $1 = 'paid' THEN NOW() ELSE paid_at END,
		    updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, status, id)
	return mapPostgresError(err)
}

func (r *PaymentPostgresRepository) Confirm(ctx context.Context, id int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	var rentalID int64
	err = tx.QueryRow(ctx, `
		UPDATE payments
		SET status = $1, paid_at = NOW(), updated_at = NOW()
		WHERE id = $2
		  AND status = 'pending'
		RETURNING rental_id
	`, models.PaymentStatusPaid, id).Scan(&rentalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.ErrInvalidPayment
		}
		return mapPostgresError(err)
	}

	result, err := tx.Exec(ctx, `
		UPDATE rentals
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		  AND status = 'pending_payment'
	`, models.RentalStatusConfirmed, rentalID)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrInvalidPayment
	}

	return mapPostgresError(tx.Commit(ctx))
}

func (r *PaymentPostgresRepository) Fail(ctx context.Context, id int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	var rentalID int64
	err = tx.QueryRow(ctx, `
		UPDATE payments
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		  AND status = 'pending'
		RETURNING rental_id
	`, models.PaymentStatusFailed, id).Scan(&rentalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.ErrInvalidPayment
		}
		return mapPostgresError(err)
	}

	if _, err = tx.Exec(ctx, `
		UPDATE rentals
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, models.RentalStatusCancelled, rentalID); err != nil {
		return mapPostgresError(err)
	}
	if _, err = tx.Exec(ctx, `
		UPDATE invoices
		SET status = $1, updated_at = NOW()
		WHERE rental_id = $2
		  AND status = 'issued'
	`, models.InvoiceStatusVoid, rentalID); err != nil {
		return mapPostgresError(err)
	}

	return mapPostgresError(tx.Commit(ctx))
}

func (r *PaymentPostgresRepository) Refund(ctx context.Context, id int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	var rentalID int64
	err = tx.QueryRow(ctx, `
		UPDATE payments
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		  AND status = 'paid'
		RETURNING rental_id
	`, models.PaymentStatusRefunded, id).Scan(&rentalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperror.ErrInvalidPayment
		}
		return mapPostgresError(err)
	}

	if _, err = tx.Exec(ctx, `
		UPDATE rentals
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		  AND status <> 'completed'
	`, models.RentalStatusCancelled, rentalID); err != nil {
		return mapPostgresError(err)
	}
	if _, err = tx.Exec(ctx, `
		UPDATE invoices
		SET status = $1, updated_at = NOW()
		WHERE rental_id = $2
		  AND status IN ('issued', 'paid')
	`, models.InvoiceStatusVoid, rentalID); err != nil {
		return mapPostgresError(err)
	}

	return mapPostgresError(tx.Commit(ctx))
}

func scanPayment(row pgx.Row) (*models.Payment, error) {
	var payment models.Payment
	if err := row.Scan(
		&payment.ID,
		&payment.RentalID,
		&payment.Amount,
		&payment.Method,
		&payment.Status,
		&payment.PaidAt,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &payment, nil
}
