package repository

import (
	"context"
	"database/sql"
	"math"
	"strconv"
	"strings"
	"time"

	"car-rental-system/internal/models"
	"car-rental-system/pkg/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RentalRepository interface {
	CreateWithAvailability(ctx context.Context, rental *models.Rental) error
	FindByID(ctx context.Context, id int64) (*models.Rental, error)
	ListCalendarRanges(ctx context.Context, carID int64, start, end time.Time) ([]AvailabilityRange, error)
	ListAll(ctx context.Context, filter RentalListFilter) (*RentalListResult, error)
	ListByUserID(ctx context.Context, userID int64) ([]models.Rental, error)
	SyncLifecycle(ctx context.Context, today time.Time) (RentalLifecycleSyncResult, error)
	UpdateStatus(ctx context.Context, id int64, status models.RentalStatus) error
	Cancel(ctx context.Context, id int64) error
}

type RentalListFilter struct {
	Status        models.RentalStatus
	PaymentStatus models.PaymentStatus
	UserID        int64
	CarID         int64
	StartFrom     time.Time
	EndTo         time.Time
	Page          int
	PageSize      int
}

type RentalListResult struct {
	Items      []models.Rental `json:"items"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

type AvailabilityRange struct {
	Source    string    `json:"source"`
	Status    string    `json:"status"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type RentalLifecycleSyncResult struct {
	Cancelled      int64
	Activated      int64
	Completed      int64
	FailedPayments int64
	VoidedInvoices int64
}

type RentalPostgresRepository struct {
	db *pgxpool.Pool
}

func NewRentalPostgresRepository(db *pgxpool.Pool) *RentalPostgresRepository {
	return &RentalPostgresRepository{db: db}
}

func (r *RentalPostgresRepository) CreateWithAvailability(ctx context.Context, rental *models.Rental) error {
	if rental.Status == "" {
		rental.Status = models.RentalStatusRequested
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	var carStatus models.CarStatus
	err = tx.QueryRow(ctx, `SELECT status FROM cars WHERE id = $1 FOR UPDATE`, rental.CarID).Scan(&carStatus)
	if err != nil {
		return mapPostgresError(err)
	}

	if carStatus == models.CarStatusMaintenance || carStatus == models.CarStatusInactive {
		return apperror.ErrCarUnavailable
	}

	var hasOverlap bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM rentals
			WHERE car_id = $1
			  AND status IN ('requested', 'approved', 'pending_payment', 'confirmed', 'active')
			  AND start_date <= $3
			  AND end_date >= $2
		)
	`, rental.CarID, rental.StartDate, rental.EndDate).Scan(&hasOverlap)
	if err != nil {
		return mapPostgresError(err)
	}
	if hasOverlap {
		return apperror.ErrDoubleBooking
	}

	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM car_maintenances
			WHERE car_id = $1
			  AND status IN ('scheduled', 'in_progress')
			  AND start_date <= $3
			  AND end_date >= $2
		)
	`, rental.CarID, rental.StartDate, rental.EndDate).Scan(&hasOverlap)
	if err != nil {
		return mapPostgresError(err)
	}
	if hasOverlap {
		return apperror.ErrCarUnavailable
	}

	query := `
		INSERT INTO rentals (user_id, car_id, start_date, end_date, total_amount, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRow(ctx, query, rental.UserID, rental.CarID, rental.StartDate, rental.EndDate, rental.TotalAmount, rental.Status).
		Scan(&rental.ID, &rental.CreatedAt, &rental.UpdatedAt)
	if err != nil {
		return mapPostgresError(err)
	}

	return mapPostgresError(tx.Commit(ctx))
}

func (r *RentalPostgresRepository) ListCalendarRanges(ctx context.Context, carID int64, start, end time.Time) ([]AvailabilityRange, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 'rental' AS source, status, start_date, end_date
		FROM rentals
		WHERE car_id = $1
		  AND status IN ('requested', 'approved', 'pending_payment', 'confirmed', 'active')
		  AND start_date <= $3
		  AND end_date >= $2
		ORDER BY start_date ASC
	`, carID, start, end)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	items := make([]AvailabilityRange, 0)
	for rows.Next() {
		var item AvailabilityRange
		if err := rows.Scan(&item.Source, &item.Status, &item.StartDate, &item.EndDate); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, mapPostgresError(rows.Err())
}

func (r *RentalPostgresRepository) FindByID(ctx context.Context, id int64) (*models.Rental, error) {
	query := rentalWithPaymentSelect + `
		WHERE r.id = $1
	`

	rental, err := scanRental(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, mapPostgresError(err)
	}

	return rental, nil
}

func (r *RentalPostgresRepository) ListAll(ctx context.Context, filter RentalListFilter) (*RentalListResult, error) {
	filter = normalizeRentalListFilter(filter)
	where, args := buildRentalListWhere(filter)

	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM rentals r
		LEFT JOIN payments p ON p.rental_id = r.id
	`+where, args...).Scan(&total); err != nil {
		return nil, mapPostgresError(err)
	}

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.Query(ctx, rentalWithPaymentSelect+where+`
		ORDER BY r.id DESC
		LIMIT $`+strconv.Itoa(limitArg)+` OFFSET $`+strconv.Itoa(offsetArg), args...)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	rentals := make([]models.Rental, 0)
	for rows.Next() {
		rental, err := scanRental(rows)
		if err != nil {
			return nil, err
		}
		rentals = append(rentals, *rental)
	}

	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(filter.PageSize)))
	}

	return &RentalListResult{
		Items:      rentals,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *RentalPostgresRepository) ListByUserID(ctx context.Context, userID int64) ([]models.Rental, error) {
	rows, err := r.db.Query(ctx, rentalWithPaymentSelect+`
		WHERE r.user_id = $1
		ORDER BY r.id DESC
	`, userID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	rentals := make([]models.Rental, 0)
	for rows.Next() {
		rental, err := scanRental(rows)
		if err != nil {
			return nil, err
		}
		rentals = append(rentals, *rental)
	}

	return rentals, mapPostgresError(rows.Err())
}

func (r *RentalPostgresRepository) SyncLifecycle(ctx context.Context, today time.Time) (RentalLifecycleSyncResult, error) {
	today = today.UTC().Truncate(24 * time.Hour)
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return RentalLifecycleSyncResult{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	var result RentalLifecycleSyncResult
	err = tx.QueryRow(ctx, `
		WITH expired_unpaid AS (
			SELECT id
			FROM rentals
			WHERE status IN ('requested', 'approved', 'pending_payment')
			  AND end_date < $1
			FOR UPDATE
		),
		failed_payments AS (
			UPDATE payments
			SET status = 'failed', updated_at = NOW()
			WHERE rental_id IN (SELECT id FROM expired_unpaid)
			  AND status = 'pending'
			RETURNING id
		),
		voided_invoices AS (
			UPDATE invoices
			SET status = 'void', updated_at = NOW()
			WHERE rental_id IN (SELECT id FROM expired_unpaid)
			  AND status = 'issued'
			RETURNING id
		),
		cancelled AS (
			UPDATE rentals
			SET status = 'cancelled', updated_at = NOW()
			WHERE id IN (SELECT id FROM expired_unpaid)
			RETURNING id
		),
		activated AS (
			UPDATE rentals
			SET status = 'active', updated_at = NOW()
			WHERE status = 'confirmed'
			  AND start_date <= $1
			  AND end_date >= $1
			RETURNING id
		),
		completed AS (
			UPDATE rentals
			SET status = 'completed', updated_at = NOW()
			WHERE status IN ('confirmed', 'active')
			  AND end_date < $1
			RETURNING id
		)
		SELECT
			(SELECT COUNT(*) FROM cancelled),
			(SELECT COUNT(*) FROM activated),
			(SELECT COUNT(*) FROM completed),
			(SELECT COUNT(*) FROM failed_payments),
			(SELECT COUNT(*) FROM voided_invoices)
	`, today).Scan(
		&result.Cancelled,
		&result.Activated,
		&result.Completed,
		&result.FailedPayments,
		&result.VoidedInvoices,
	)
	if err != nil {
		return RentalLifecycleSyncResult{}, mapPostgresError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return RentalLifecycleSyncResult{}, mapPostgresError(err)
	}
	return result, nil
}

func (r *RentalPostgresRepository) UpdateStatus(ctx context.Context, id int64, status models.RentalStatus) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `UPDATE rentals SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}

	if status == models.RentalStatusCancelled || status == models.RentalStatusRejected {
		if _, err = tx.Exec(ctx, `
			UPDATE invoices
			SET status = $1, updated_at = NOW()
			WHERE rental_id = $2
			  AND status = 'issued'
		`, models.InvoiceStatusVoid, id); err != nil {
			return mapPostgresError(err)
		}
	}

	return mapPostgresError(tx.Commit(ctx))
}

func (r *RentalPostgresRepository) Cancel(ctx context.Context, id int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `UPDATE rentals SET status = $1, updated_at = NOW() WHERE id = $2`, models.RentalStatusCancelled, id)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}

	if _, err = tx.Exec(ctx, `
		UPDATE invoices
		SET status = $1, updated_at = NOW()
		WHERE rental_id = $2
		  AND status = 'issued'
	`, models.InvoiceStatusVoid, id); err != nil {
		return mapPostgresError(err)
	}

	return mapPostgresError(tx.Commit(ctx))
}

const rentalWithPaymentSelect = `
	SELECT
		r.id, r.user_id, r.car_id, r.start_date, r.end_date, r.total_amount, r.status, r.created_at, r.updated_at,
		p.id, p.rental_id, p.amount, p.method, p.status, p.paid_at, p.created_at, p.updated_at,
		u.id, u.name, u.email, u.phone, u.role, u.email_verified_at, u.created_at, u.updated_at,
		c.id, c.brand, c.model, c.year, c.plate_number, c.daily_rate, c.seats, c.fuel, c.transmission, c.status, COALESCE(c.image_url, ''), c.created_at, c.updated_at
	FROM rentals r
	JOIN users u ON u.id = r.user_id
	JOIN cars c ON c.id = r.car_id
	LEFT JOIN payments p ON p.rental_id = r.id
`

func scanRental(row pgx.Row) (*models.Rental, error) {
	var rental models.Rental
	var user models.User
	var car models.Car
	var paymentID sql.NullInt64
	var paymentRentalID sql.NullInt64
	var paymentAmount sql.NullFloat64
	var paymentMethod sql.NullString
	var paymentStatus sql.NullString
	var paymentPaidAt sql.NullTime
	var paymentCreatedAt sql.NullTime
	var paymentUpdatedAt sql.NullTime
	var userEmailVerifiedAt sql.NullTime

	if err := row.Scan(
		&rental.ID,
		&rental.UserID,
		&rental.CarID,
		&rental.StartDate,
		&rental.EndDate,
		&rental.TotalAmount,
		&rental.Status,
		&rental.CreatedAt,
		&rental.UpdatedAt,
		&paymentID,
		&paymentRentalID,
		&paymentAmount,
		&paymentMethod,
		&paymentStatus,
		&paymentPaidAt,
		&paymentCreatedAt,
		&paymentUpdatedAt,
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Role,
		&userEmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&car.ID,
		&car.Brand,
		&car.Model,
		&car.Year,
		&car.PlateNumber,
		&car.DailyRate,
		&car.Seats,
		&car.Fuel,
		&car.Transmission,
		&car.Status,
		&car.Image,
		&car.CreatedAt,
		&car.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if userEmailVerifiedAt.Valid {
		user.EmailVerifiedAt = &userEmailVerifiedAt.Time
	}
	rental.User = &user
	rental.Car = &car

	if paymentID.Valid {
		var paidAt *time.Time
		if paymentPaidAt.Valid {
			paidAt = &paymentPaidAt.Time
		}

		rental.Payment = &models.Payment{
			ID:        paymentID.Int64,
			RentalID:  paymentRentalID.Int64,
			Amount:    paymentAmount.Float64,
			Method:    models.PaymentMethod(paymentMethod.String),
			Status:    models.PaymentStatus(paymentStatus.String),
			PaidAt:    paidAt,
			CreatedAt: paymentCreatedAt.Time,
			UpdatedAt: paymentUpdatedAt.Time,
		}
	}

	return &rental, nil
}

func normalizeRentalListFilter(filter RentalListFilter) RentalListFilter {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	return filter
}

func buildRentalListWhere(filter RentalListFilter) (string, []any) {
	conditions := make([]string, 0)
	args := make([]any, 0)

	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, "r.status = $"+strconv.Itoa(len(args)))
	}
	if filter.PaymentStatus != "" {
		args = append(args, filter.PaymentStatus)
		conditions = append(conditions, "p.status = $"+strconv.Itoa(len(args)))
	}
	if filter.UserID > 0 {
		args = append(args, filter.UserID)
		conditions = append(conditions, "r.user_id = $"+strconv.Itoa(len(args)))
	}
	if filter.CarID > 0 {
		args = append(args, filter.CarID)
		conditions = append(conditions, "r.car_id = $"+strconv.Itoa(len(args)))
	}
	if !filter.StartFrom.IsZero() {
		args = append(args, filter.StartFrom)
		conditions = append(conditions, "r.start_date >= $"+strconv.Itoa(len(args)))
	}
	if !filter.EndTo.IsZero() {
		args = append(args, filter.EndTo)
		conditions = append(conditions, "r.end_date <= $"+strconv.Itoa(len(args)))
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}
