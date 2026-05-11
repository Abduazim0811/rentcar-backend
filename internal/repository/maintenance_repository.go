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

type MaintenanceRepository interface {
	Create(ctx context.Context, item *models.CarMaintenance) error
	Update(ctx context.Context, item *models.CarMaintenance) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*models.CarMaintenance, error)
	List(ctx context.Context, filter MaintenanceListFilter) (*MaintenanceListResult, error)
	ListCalendarRanges(ctx context.Context, carID int64, start, end time.Time) ([]AvailabilityRange, error)
}

type MaintenanceListFilter struct {
	CarID    int64
	Status   models.MaintenanceStatus
	Page     int
	PageSize int
}

type MaintenanceListResult struct {
	Items      []models.CarMaintenance `json:"items"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

type MaintenancePostgresRepository struct {
	db *pgxpool.Pool
}

func NewMaintenancePostgresRepository(db *pgxpool.Pool) *MaintenancePostgresRepository {
	return &MaintenancePostgresRepository{db: db}
}

func (r *MaintenancePostgresRepository) Create(ctx context.Context, item *models.CarMaintenance) error {
	if item.Status == "" {
		item.Status = models.MaintenanceStatusScheduled
	}
	query := `
		INSERT INTO car_maintenances (car_id, start_date, end_date, reason, status, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, item.CarID, item.StartDate, item.EndDate, item.Reason, item.Status, item.Notes, item.CreatedBy).
		Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return mapPostgresError(err)
}

func (r *MaintenancePostgresRepository) Update(ctx context.Context, item *models.CarMaintenance) error {
	query := `
		UPDATE car_maintenances
		SET car_id = $1, start_date = $2, end_date = $3, reason = $4, status = $5, notes = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING created_by, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query, item.CarID, item.StartDate, item.EndDate, item.Reason, item.Status, item.Notes, item.ID).
		Scan(&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	return mapPostgresError(err)
}

func (r *MaintenancePostgresRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM car_maintenances WHERE id = $1`, id)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *MaintenancePostgresRepository) FindByID(ctx context.Context, id int64) (*models.CarMaintenance, error) {
	item, err := scanMaintenance(r.db.QueryRow(ctx, `
		SELECT id, car_id, start_date, end_date, reason, status, notes, created_by, created_at, updated_at
		FROM car_maintenances
		WHERE id = $1
	`, id))
	if err != nil {
		return nil, mapPostgresError(err)
	}
	return item, nil
}

func (r *MaintenancePostgresRepository) List(ctx context.Context, filter MaintenanceListFilter) (*MaintenanceListResult, error) {
	filter = normalizeMaintenanceListFilter(filter)
	where, args := buildMaintenanceWhere(filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM car_maintenances`+where, args...).Scan(&total); err != nil {
		return nil, mapPostgresError(err)
	}

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.Query(ctx, `
		SELECT id, car_id, start_date, end_date, reason, status, notes, created_by, created_at, updated_at
		FROM car_maintenances
	`+where+`
		ORDER BY start_date DESC, id DESC
		LIMIT $`+strconv.Itoa(limitArg)+` OFFSET $`+strconv.Itoa(offsetArg), args...)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	items := make([]models.CarMaintenance, 0)
	for rows.Next() {
		item, err := scanMaintenance(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(filter.PageSize)))
	}

	return &MaintenanceListResult{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize, TotalPages: totalPages}, nil
}

func (r *MaintenancePostgresRepository) ListCalendarRanges(ctx context.Context, carID int64, start, end time.Time) ([]AvailabilityRange, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 'maintenance' AS source, status, start_date, end_date
		FROM car_maintenances
		WHERE car_id = $1
		  AND status IN ('scheduled', 'in_progress')
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

func normalizeMaintenanceListFilter(filter MaintenanceListFilter) MaintenanceListFilter {
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

func buildMaintenanceWhere(filter MaintenanceListFilter) (string, []any) {
	args := make([]any, 0)
	conditions := make([]string, 0)
	if filter.CarID > 0 {
		args = append(args, filter.CarID)
		conditions = append(conditions, "car_id = $"+strconv.Itoa(len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, "status = $"+strconv.Itoa(len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanMaintenance(row pgx.Row) (*models.CarMaintenance, error) {
	var item models.CarMaintenance
	var createdBy sql.NullInt64
	if err := row.Scan(&item.ID, &item.CarID, &item.StartDate, &item.EndDate, &item.Reason, &item.Status, &item.Notes, &createdBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if createdBy.Valid {
		item.CreatedBy = &createdBy.Int64
	}
	return &item, nil
}
