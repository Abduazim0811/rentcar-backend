package service

import (
	"context"
	"errors"

	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
	"car-rental-system/pkg/apperror"
)

type PaymentService struct {
	payments repository.PaymentRepository
	rentals  repository.RentalRepository
}

type PaymentInput struct {
	RentalID int64                `json:"rental_id" binding:"required"`
	Method   models.PaymentMethod `json:"method" binding:"required,oneof=cash card bank_transfer"`
}

func NewPaymentService(payments repository.PaymentRepository, rentals repository.RentalRepository) *PaymentService {
	return &PaymentService{payments: payments, rentals: rentals}
}

func (s *PaymentService) Create(ctx context.Context, userID int64, role models.UserRole, input PaymentInput) (*models.Payment, error) {
	rental, err := s.rentals.FindByID(ctx, input.RentalID)
	if err != nil {
		return nil, err
	}
	if rental.UserID != userID && role != models.RoleAdmin && role != models.RoleSuperAdmin {
		return nil, apperror.ErrForbidden
	}
	if rental.Status != models.RentalStatusPendingPayment && rental.Status != models.RentalStatusApproved {
		return nil, apperror.New(400, "payment can be created only for approved rentals")
	}
	if _, err := s.payments.FindByRentalID(ctx, rental.ID); err == nil {
		return nil, apperror.ErrPaymentExists
	} else if !errors.Is(err, apperror.ErrNotFound) {
		return nil, err
	}

	payment := &models.Payment{
		RentalID: rental.ID,
		Amount:   rental.TotalAmount,
		Method:   input.Method,
		Status:   models.PaymentStatusPending,
	}

	if err := s.payments.Create(ctx, payment); err != nil {
		return nil, err
	}
	if rental.Status == models.RentalStatusApproved {
		if err := s.rentals.UpdateStatus(ctx, rental.ID, models.RentalStatusPendingPayment); err != nil {
			return nil, err
		}
	}

	return payment, nil
}

func (s *PaymentService) Confirm(ctx context.Context, id int64) error {
	return s.payments.Confirm(ctx, id)
}

func (s *PaymentService) Fail(ctx context.Context, id int64) error {
	return s.payments.Fail(ctx, id)
}

func (s *PaymentService) Refund(ctx context.Context, id int64) error {
	return s.payments.Refund(ctx, id)
}

func (s *PaymentService) FindByID(ctx context.Context, id int64) (*models.Payment, error) {
	return s.payments.FindByID(ctx, id)
}
