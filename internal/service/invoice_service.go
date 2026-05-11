package service

import (
	"context"
	"fmt"
	"time"

	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
	"car-rental-system/pkg/apperror"
)

type InvoiceService struct {
	invoices repository.InvoiceRepository
	rentals  repository.RentalRepository
}

func NewInvoiceService(invoices repository.InvoiceRepository, rentals repository.RentalRepository) *InvoiceService {
	return &InvoiceService{invoices: invoices, rentals: rentals}
}

func (s *InvoiceService) GetOrCreate(ctx context.Context, userID int64, role models.UserRole, rentalID int64) (*models.Invoice, error) {
	rental, err := s.rentals.FindByID(ctx, rentalID)
	if err != nil {
		return nil, err
	}
	if rental.UserID != userID && role != models.RoleAdmin && role != models.RoleSuperAdmin {
		return nil, apperror.ErrForbidden
	}

	invoice, err := s.invoices.FindByRentalID(ctx, rentalID)
	if err == nil {
		return invoice, nil
	}
	if err != apperror.ErrNotFound {
		return nil, err
	}

	now := time.Now().UTC()
	invoice = &models.Invoice{
		RentalID:      rental.ID,
		InvoiceNumber: fmt.Sprintf("INV-%d-%06d", now.Year(), rental.ID),
		Amount:        rental.TotalAmount,
		Status:        models.InvoiceStatusIssued,
		IssuedAt:      now,
	}
	if rental.Payment != nil && rental.Payment.Status == models.PaymentStatusPaid {
		invoice.Status = models.InvoiceStatusPaid
		invoice.PaidAt = rental.Payment.PaidAt
	}
	if err := s.invoices.Create(ctx, invoice); err != nil {
		return nil, err
	}
	return invoice, nil
}

func (s *InvoiceService) MarkPaid(ctx context.Context, rentalID int64) error {
	return s.invoices.MarkPaid(ctx, rentalID)
}
