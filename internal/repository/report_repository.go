package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepository interface {
	Summary(ctx context.Context, start, end time.Time) (*ReportSummary, error)
}

type ReportSummary struct {
	StartDate       string        `json:"start_date"`
	EndDate         string        `json:"end_date"`
	Revenue         float64       `json:"revenue"`
	TotalRentals    int64         `json:"total_rentals"`
	ActiveRentals   int64         `json:"active_rentals"`
	PendingPayments int64         `json:"pending_payments"`
	Cancelled       int64         `json:"cancelled"`
	TopCars         []TopCarStats `json:"top_cars"`
}

type TopCarStats struct {
	CarID       int64   `json:"car_id"`
	Label       string  `json:"label"`
	RentalCount int64   `json:"rental_count"`
	Revenue     float64 `json:"revenue"`
}

type ReportPostgresRepository struct {
	db *pgxpool.Pool
}

func NewReportPostgresRepository(db *pgxpool.Pool) *ReportPostgresRepository {
	return &ReportPostgresRepository{db: db}
}

func (r *ReportPostgresRepository) Summary(ctx context.Context, start, end time.Time) (*ReportSummary, error) {
	result := &ReportSummary{
		StartDate: start.Format(time.DateOnly),
		EndDate:   end.Format(time.DateOnly),
		TopCars:   make([]TopCarStats, 0),
	}

	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN p.status = 'paid' THEN p.amount ELSE 0 END), 0),
			COUNT(DISTINCT rentals.id),
			COUNT(DISTINCT rentals.id) FILTER (WHERE rentals.status IN ('confirmed', 'active')),
			COUNT(DISTINCT p.id) FILTER (WHERE p.status = 'pending'),
			COUNT(DISTINCT rentals.id) FILTER (WHERE rentals.status IN ('cancelled', 'rejected'))
		FROM rentals
		LEFT JOIN payments p ON p.rental_id = rentals.id
		WHERE rentals.created_at >= $1 AND rentals.created_at < $2
	`, start, end.Add(24*time.Hour)).Scan(&result.Revenue, &result.TotalRentals, &result.ActiveRentals, &result.PendingPayments, &result.Cancelled)
	if err != nil {
		return nil, mapPostgresError(err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.brand || ' ' || c.model AS label, COUNT(r.id) AS rental_count, COALESCE(SUM(CASE WHEN p.status = 'paid' THEN p.amount ELSE 0 END), 0) AS revenue
		FROM rentals r
		JOIN cars c ON c.id = r.car_id
		LEFT JOIN payments p ON p.rental_id = r.id
		WHERE r.created_at >= $1 AND r.created_at < $2
		GROUP BY c.id, c.brand, c.model
		ORDER BY rental_count DESC, revenue DESC
		LIMIT 5
	`, start, end.Add(24*time.Hour))
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var item TopCarStats
		if err := rows.Scan(&item.CarID, &item.Label, &item.RentalCount, &item.Revenue); err != nil {
			return nil, err
		}
		result.TopCars = append(result.TopCars, item)
	}

	return result, mapPostgresError(rows.Err())
}
