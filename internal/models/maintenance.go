package models

import "time"

type MaintenanceStatus string

const (
	MaintenanceStatusScheduled  MaintenanceStatus = "scheduled"
	MaintenanceStatusInProgress MaintenanceStatus = "in_progress"
	MaintenanceStatusCompleted  MaintenanceStatus = "completed"
	MaintenanceStatusCancelled  MaintenanceStatus = "cancelled"
)

type CarMaintenance struct {
	ID        int64             `json:"id"`
	CarID     int64             `json:"car_id"`
	StartDate time.Time         `json:"start_date"`
	EndDate   time.Time         `json:"end_date"`
	Reason    string            `json:"reason"`
	Status    MaintenanceStatus `json:"status"`
	Notes     string            `json:"notes"`
	CreatedBy *int64            `json:"created_by,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}
