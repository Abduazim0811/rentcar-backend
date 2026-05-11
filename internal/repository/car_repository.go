package repository

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"car-rental-system/internal/models"
	"car-rental-system/pkg/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CarRepository interface {
	Create(ctx context.Context, car *models.Car) error
	Update(ctx context.Context, car *models.Car) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*models.Car, error)
	List(ctx context.Context, filter CarListFilter) (*CarListResult, error)
	IsAvailable(ctx context.Context, carID int64, startDate, endDate time.Time) (bool, error)
}

type CarListFilter struct {
	Search       string
	Status       models.CarStatus
	MinYear      int
	MaxYear      int
	MinDailyRate float64
	MaxDailyRate float64
	Page         int
	PageSize     int
}

type CarListResult struct {
	Items      []models.Car `json:"items"`
	Total      int64        `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalPages int          `json:"total_pages"`
}

type CarPostgresRepository struct {
	db *pgxpool.Pool
}

func NewCarPostgresRepository(db *pgxpool.Pool) *CarPostgresRepository {
	return &CarPostgresRepository{db: db}
}

func (r *CarPostgresRepository) Create(ctx context.Context, car *models.Car) error {
	query := `
		INSERT INTO cars (brand, model, year, plate_number, daily_rate, status, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	if car.Status == "" {
		car.Status = models.CarStatusAvailable
	}

	err := r.db.QueryRow(ctx, query, car.Brand, car.Model, car.Year, car.PlateNumber, car.DailyRate, car.Status, car.Image).
		Scan(&car.ID, &car.CreatedAt, &car.UpdatedAt)
	return mapPostgresError(err)
}

func (r *CarPostgresRepository) Update(ctx context.Context, car *models.Car) error {
	query := `
		UPDATE cars
		SET brand = $1, model = $2, year = $3, plate_number = $4, daily_rate = $5, status = $6, image_url = $7, updated_at = NOW()
		WHERE id = $8
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, car.Brand, car.Model, car.Year, car.PlateNumber, car.DailyRate, car.Status, car.Image, car.ID).
		Scan(&car.CreatedAt, &car.UpdatedAt)
	return mapPostgresError(err)
}

func (r *CarPostgresRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM cars WHERE id = $1`, id)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}

	return nil
}

func (r *CarPostgresRepository) FindByID(ctx context.Context, id int64) (*models.Car, error) {
	query := `
		SELECT id, brand, model, year, plate_number, daily_rate, status, COALESCE(image_url, ''), created_at, updated_at
		FROM cars
		WHERE id = $1
	`

	car, err := scanCar(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, mapPostgresError(err)
	}

	return car, nil
}

func (r *CarPostgresRepository) List(ctx context.Context, filter CarListFilter) (*CarListResult, error) {
	filter = normalizeCarListFilter(filter)
	where, args := buildCarListWhere(filter)

	var total int64
	countQuery := `SELECT COUNT(*) FROM cars` + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, mapPostgresError(err)
	}

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	query := `
		SELECT id, brand, model, year, plate_number, daily_rate, status, COALESCE(image_url, ''), created_at, updated_at
		FROM cars
	` + where + `
		ORDER BY id DESC
		LIMIT $` + strconv.Itoa(limitArg) + ` OFFSET $` + strconv.Itoa(offsetArg)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	cars := make([]models.Car, 0)
	for rows.Next() {
		car, err := scanCar(rows)
		if err != nil {
			return nil, err
		}
		cars = append(cars, *car)
	}

	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(filter.PageSize)))
	}

	return &CarListResult{
		Items:      cars,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *CarPostgresRepository) IsAvailable(ctx context.Context, carID int64, startDate, endDate time.Time) (bool, error) {
	query := `
		SELECT NOT EXISTS (
			SELECT 1
			FROM rentals
	WHERE car_id = $1
			  AND status IN ('requested', 'approved', 'pending_payment', 'confirmed', 'active')
			  AND start_date <= $3
			  AND end_date >= $2
		)
		AND NOT EXISTS (
			SELECT 1
			FROM car_maintenances
			WHERE car_id = $1
			  AND status IN ('scheduled', 'in_progress')
			  AND start_date <= $3
			  AND end_date >= $2
		)
	`

	var available bool
	err := r.db.QueryRow(ctx, query, carID, startDate, endDate).Scan(&available)
	return available, mapPostgresError(err)
}

func normalizeCarListFilter(filter CarListFilter) CarListFilter {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 12
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	filter.Search = strings.TrimSpace(filter.Search)
	return filter
}

func buildCarListWhere(filter CarListFilter) (string, []any) {
	conditions := make([]string, 0)
	args := make([]any, 0)

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		placeholder := "$" + strconv.Itoa(len(args))
		conditions = append(conditions, "(brand ILIKE "+placeholder+" OR model ILIKE "+placeholder+" OR plate_number ILIKE "+placeholder+" OR CAST(year AS TEXT) ILIKE "+placeholder+")")
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, "status = $"+strconv.Itoa(len(args)))
	}
	if filter.MinYear > 0 {
		args = append(args, filter.MinYear)
		conditions = append(conditions, "year >= $"+strconv.Itoa(len(args)))
	}
	if filter.MaxYear > 0 {
		args = append(args, filter.MaxYear)
		conditions = append(conditions, "year <= $"+strconv.Itoa(len(args)))
	}
	if filter.MinDailyRate > 0 {
		args = append(args, filter.MinDailyRate)
		conditions = append(conditions, "daily_rate >= $"+strconv.Itoa(len(args)))
	}
	if filter.MaxDailyRate > 0 {
		args = append(args, filter.MaxDailyRate)
		conditions = append(conditions, "daily_rate <= $"+strconv.Itoa(len(args)))
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanCar(row pgx.Row) (*models.Car, error) {
	var car models.Car
	if err := row.Scan(
		&car.ID,
		&car.Brand,
		&car.Model,
		&car.Year,
		&car.PlateNumber,
		&car.DailyRate,
		&car.Status,
		&car.Image,
		&car.CreatedAt,
		&car.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &car, nil
}
