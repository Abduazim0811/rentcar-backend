package models

import "time"

type InvoiceStatus string

const (
	InvoiceStatusIssued InvoiceStatus = "issued"
	InvoiceStatusPaid   InvoiceStatus = "paid"
	InvoiceStatusVoid   InvoiceStatus = "void"
)

type Invoice struct {
	ID            int64         `json:"id"`
	RentalID      int64         `json:"rental_id"`
	InvoiceNumber string        `json:"invoice_number"`
	Amount        float64       `json:"amount"`
	Status        InvoiceStatus `json:"status"`
	IssuedAt      time.Time     `json:"issued_at"`
	DueAt         *time.Time    `json:"due_at"`
	PaidAt        *time.Time    `json:"paid_at"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}
