package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
	"car-rental-system/pkg/apperror"
)

type paymentServicePaymentRepo struct {
	existing      *models.Payment
	findErr       error
	created       *models.Payment
	createErr     error
	confirmCalled bool
	failCalled    bool
	refundCalled  bool
}

func (r *paymentServicePaymentRepo) Create(_ context.Context, payment *models.Payment) error {
	if r.createErr != nil {
		return r.createErr
	}
	copy := *payment
	copy.ID = 55
	r.created = &copy
	payment.ID = copy.ID
	return nil
}
func (r *paymentServicePaymentRepo) FindByID(context.Context, int64) (*models.Payment, error) {
	return nil, apperror.ErrNotFound
}
func (r *paymentServicePaymentRepo) FindByRentalID(context.Context, int64) (*models.Payment, error) {
	if r.existing != nil {
		return r.existing, nil
	}
	if r.findErr != nil {
		return nil, r.findErr
	}
	return nil, apperror.ErrNotFound
}
func (r *paymentServicePaymentRepo) UpdateStatus(context.Context, int64, models.PaymentStatus) error {
	return nil
}
func (r *paymentServicePaymentRepo) Confirm(context.Context, int64) error {
	r.confirmCalled = true
	return nil
}
func (r *paymentServicePaymentRepo) Fail(context.Context, int64) error {
	r.failCalled = true
	return nil
}
func (r *paymentServicePaymentRepo) Refund(context.Context, int64) error {
	r.refundCalled = true
	return nil
}

type paymentServiceRentalRepo struct {
	rental *models.Rental
}

func (r *paymentServiceRentalRepo) CreateWithAvailability(context.Context, *models.Rental) error {
	return nil
}
func (r *paymentServiceRentalRepo) FindByID(context.Context, int64) (*models.Rental, error) {
	if r.rental == nil {
		return nil, apperror.ErrNotFound
	}
	return r.rental, nil
}
func (r *paymentServiceRentalRepo) ListCalendarRanges(context.Context, int64, time.Time, time.Time) ([]repository.AvailabilityRange, error) {
	return nil, nil
}
func (r *paymentServiceRentalRepo) ListAll(context.Context, repository.RentalListFilter) (*repository.RentalListResult, error) {
	return nil, nil
}
func (r *paymentServiceRentalRepo) ListByUserID(context.Context, int64) ([]models.Rental, error) {
	return nil, nil
}
func (r *paymentServiceRentalRepo) UpdateStatus(context.Context, int64, models.RentalStatus) error {
	return nil
}
func (r *paymentServiceRentalRepo) Cancel(context.Context, int64) error { return nil }

func TestPaymentCreateRejectsDuplicatePayment(t *testing.T) {
	paymentRepo := &paymentServicePaymentRepo{existing: &models.Payment{ID: 1, RentalID: 10}}
	rentalRepo := &paymentServiceRentalRepo{rental: &models.Rental{
		ID:          10,
		UserID:      7,
		TotalAmount: 120,
		Status:      models.RentalStatusPendingPayment,
	}}
	service := NewPaymentService(paymentRepo, rentalRepo)

	_, err := service.Create(context.Background(), 7, models.RoleCustomer, PaymentInput{
		RentalID: 10,
		Method:   models.PaymentMethodCash,
	})

	if !errors.Is(err, apperror.ErrPaymentExists) {
		t.Fatalf("expected payment exists error, got %v", err)
	}
	if paymentRepo.created != nil {
		t.Fatal("duplicate payment should not create a new payment")
	}
}

func TestPaymentCreateRejectsDifferentCustomer(t *testing.T) {
	service := NewPaymentService(&paymentServicePaymentRepo{}, &paymentServiceRentalRepo{rental: &models.Rental{
		ID:          10,
		UserID:      7,
		TotalAmount: 120,
		Status:      models.RentalStatusPendingPayment,
	}})

	_, err := service.Create(context.Background(), 8, models.RoleCustomer, PaymentInput{
		RentalID: 10,
		Method:   models.PaymentMethodCash,
	})

	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestPaymentCreateUsesRentalAmountAndPendingStatus(t *testing.T) {
	paymentRepo := &paymentServicePaymentRepo{}
	service := NewPaymentService(paymentRepo, &paymentServiceRentalRepo{rental: &models.Rental{
		ID:          10,
		UserID:      7,
		TotalAmount: 120,
		Status:      models.RentalStatusPendingPayment,
	}})

	payment, err := service.Create(context.Background(), 7, models.RoleCustomer, PaymentInput{
		RentalID: 10,
		Method:   models.PaymentMethodBankTransfer,
	})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	if payment.ID != 55 {
		t.Fatalf("expected repository-assigned id 55, got %d", payment.ID)
	}
	if payment.Amount != 120 {
		t.Fatalf("expected amount 120, got %v", payment.Amount)
	}
	if payment.Status != models.PaymentStatusPending {
		t.Fatalf("expected pending payment, got %s", payment.Status)
	}
	if payment.Method != models.PaymentMethodBankTransfer {
		t.Fatalf("expected bank transfer method, got %s", payment.Method)
	}
}

func TestPaymentRefundDelegatesToRepository(t *testing.T) {
	paymentRepo := &paymentServicePaymentRepo{}
	service := NewPaymentService(paymentRepo, &paymentServiceRentalRepo{})

	if err := service.Refund(context.Background(), 55); err != nil {
		t.Fatalf("refund payment: %v", err)
	}
	if !paymentRepo.refundCalled {
		t.Fatal("expected refund repository method to be called")
	}
}
