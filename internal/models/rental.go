package models

import "time"

type RentalStatus string

const (
	RentalStatusRequested      RentalStatus = "requested"
	RentalStatusApproved       RentalStatus = "approved"
	RentalStatusRejected       RentalStatus = "rejected"
	RentalStatusPendingPayment RentalStatus = "pending_payment"
	RentalStatusConfirmed      RentalStatus = "confirmed"
	RentalStatusActive         RentalStatus = "active"
	RentalStatusCancelled      RentalStatus = "cancelled"
	RentalStatusCompleted      RentalStatus = "completed"
)

type Rental struct {
	ID          int64        `json:"id"`
	UserID      int64        `json:"user_id"`
	CarID       int64        `json:"car_id"`
	StartDate   time.Time    `json:"start_date"`
	EndDate     time.Time    `json:"end_date"`
	TotalAmount float64      `json:"total_amount"`
	Status      RentalStatus `json:"status"`
	Payment     *Payment     `json:"payment,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}
