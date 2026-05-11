package models

import "time"

type CarStatus string

const (
	CarStatusAvailable   CarStatus = "available"
	CarStatusRented      CarStatus = "rented"
	CarStatusMaintenance CarStatus = "maintenance"
	CarStatusInactive    CarStatus = "inactive"
)

type Car struct {
	ID          int64     `json:"id"`
	Brand       string    `json:"brand"`
	Model       string    `json:"model"`
	Year        int       `json:"year"`
	PlateNumber string    `json:"plate_number"`
	DailyRate   float64   `json:"daily_rate"`
	Status      CarStatus `json:"status"`
	Image       string    `json:"image,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
