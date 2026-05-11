package models

import "time"

type PaymentStatus string
type PaymentMethod string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusRefunded PaymentStatus = "refunded"

	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodCard         PaymentMethod = "card"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
)

type Payment struct {
	ID        int64         `json:"id"`
	RentalID  int64         `json:"rental_id"`
	Amount    float64       `json:"amount"`
	Method    PaymentMethod `json:"method"`
	Status    PaymentStatus `json:"status"`
	PaidAt    *time.Time    `json:"paid_at"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
