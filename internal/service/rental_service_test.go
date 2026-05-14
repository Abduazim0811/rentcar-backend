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

type rentalServiceCarRepo struct {
	car             *models.Car
	available       bool
	createWasCalled bool
}

func (r *rentalServiceCarRepo) Create(context.Context, *models.Car) error { return nil }
func (r *rentalServiceCarRepo) Update(context.Context, *models.Car) error { return nil }
func (r *rentalServiceCarRepo) Delete(context.Context, int64) error       { return nil }
func (r *rentalServiceCarRepo) FindByID(context.Context, int64) (*models.Car, error) {
	if r.car == nil {
		return nil, apperror.ErrNotFound
	}
	return r.car, nil
}
func (r *rentalServiceCarRepo) List(context.Context, repository.CarListFilter) (*repository.CarListResult, error) {
	return nil, nil
}
func (r *rentalServiceCarRepo) IsAvailable(context.Context, int64, time.Time, time.Time) (bool, error) {
	return r.available, nil
}

type rentalServiceRentalRepo struct {
	rental        *models.Rental
	created       *models.Rental
	cancelledID   int64
	cancelCalled  bool
	updatedID     int64
	updatedStatus models.RentalStatus
	updateCalled  bool
}

func (r *rentalServiceRentalRepo) CreateWithAvailability(_ context.Context, rental *models.Rental) error {
	r.createCopy(rental)
	return nil
}
func (r *rentalServiceRentalRepo) createCopy(rental *models.Rental) {
	copy := *rental
	copy.ID = 99
	r.created = &copy
	rental.ID = copy.ID
}
func (r *rentalServiceRentalRepo) FindByID(context.Context, int64) (*models.Rental, error) {
	if r.rental == nil {
		return nil, apperror.ErrNotFound
	}
	return r.rental, nil
}
func (r *rentalServiceRentalRepo) ListCalendarRanges(context.Context, int64, time.Time, time.Time) ([]repository.AvailabilityRange, error) {
	return nil, nil
}
func (r *rentalServiceRentalRepo) ListAll(context.Context, repository.RentalListFilter) (*repository.RentalListResult, error) {
	return nil, nil
}
func (r *rentalServiceRentalRepo) ListByUserID(context.Context, int64) ([]models.Rental, error) {
	return nil, nil
}
func (r *rentalServiceRentalRepo) SyncLifecycle(context.Context, time.Time) (repository.RentalLifecycleSyncResult, error) {
	return repository.RentalLifecycleSyncResult{}, nil
}
func (r *rentalServiceRentalRepo) UpdateStatus(_ context.Context, id int64, status models.RentalStatus) error {
	r.updateCalled = true
	r.updatedID = id
	r.updatedStatus = status
	return nil
}
func (r *rentalServiceRentalRepo) Cancel(_ context.Context, id int64) error {
	r.cancelCalled = true
	r.cancelledID = id
	return nil
}

func TestRentalCreatePreventsDoubleBooking(t *testing.T) {
	carRepo := &rentalServiceCarRepo{
		car:       &models.Car{ID: 10, Status: models.CarStatusAvailable, DailyRate: 50},
		available: false,
	}
	rentalRepo := &rentalServiceRentalRepo{}
	service := NewRentalService(rentalRepo, carRepo)
	start := time.Now().UTC().AddDate(0, 0, 1).Format(time.DateOnly)
	end := time.Now().UTC().AddDate(0, 0, 3).Format(time.DateOnly)

	_, err := service.Create(context.Background(), 7, RentalInput{
		CarID:     10,
		StartDate: start,
		EndDate:   end,
	})

	if !errors.Is(err, apperror.ErrDoubleBooking) {
		t.Fatalf("expected double booking error, got %v", err)
	}
	if rentalRepo.created != nil {
		t.Fatal("rental should not be created when car is unavailable")
	}
}

func TestRentalCreateCalculatesTotalAndCreatesPendingRental(t *testing.T) {
	carRepo := &rentalServiceCarRepo{
		car:       &models.Car{ID: 10, Status: models.CarStatusAvailable, DailyRate: 40},
		available: true,
	}
	rentalRepo := &rentalServiceRentalRepo{}
	service := NewRentalService(rentalRepo, carRepo)
	start := time.Now().UTC().AddDate(0, 0, 1).Format(time.DateOnly)
	end := time.Now().UTC().AddDate(0, 0, 3).Format(time.DateOnly)

	rental, err := service.Create(context.Background(), 7, RentalInput{
		CarID:     10,
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		t.Fatalf("create rental: %v", err)
	}

	if rental.ID != 99 {
		t.Fatalf("expected repository-assigned id, got %d", rental.ID)
	}
	if rental.TotalAmount != 120 {
		t.Fatalf("expected total 120, got %v", rental.TotalAmount)
	}
	if rental.Status != models.RentalStatusRequested {
		t.Fatalf("expected requested status, got %s", rental.Status)
	}
}

func TestRentalCreateRejectsPastStartDate(t *testing.T) {
	carRepo := &rentalServiceCarRepo{
		car:       &models.Car{ID: 10, Status: models.CarStatusAvailable, DailyRate: 40},
		available: true,
	}
	rentalRepo := &rentalServiceRentalRepo{}
	service := NewRentalService(rentalRepo, carRepo)
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format(time.DateOnly)
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format(time.DateOnly)

	_, err := service.Create(context.Background(), 7, RentalInput{
		CarID:     10,
		StartDate: yesterday,
		EndDate:   tomorrow,
	})

	if err == nil {
		t.Fatal("expected past date error")
	}
	if rentalRepo.created != nil {
		t.Fatal("rental should not be created for a past start date")
	}
}

func TestRentalUpdateStatusBlocksActiveWithoutPaidPayment(t *testing.T) {
	rentalRepo := &rentalServiceRentalRepo{rental: &models.Rental{
		ID:     44,
		UserID: 7,
		Status: models.RentalStatusConfirmed,
	}}
	service := NewRentalService(rentalRepo, &rentalServiceCarRepo{})

	err := service.UpdateStatus(context.Background(), 44, models.RentalStatusActive)
	if err == nil {
		t.Fatal("expected payment policy error")
	}
	if rentalRepo.updateCalled {
		t.Fatal("repository update should not be called")
	}
}

func TestRentalUpdateStatusRequiresPaymentRequestForPendingPayment(t *testing.T) {
	rentalRepo := &rentalServiceRentalRepo{rental: &models.Rental{
		ID:     44,
		UserID: 7,
		Status: models.RentalStatusApproved,
	}}
	service := NewRentalService(rentalRepo, &rentalServiceCarRepo{})

	err := service.UpdateStatus(context.Background(), 44, models.RentalStatusPendingPayment)
	if err == nil {
		t.Fatal("expected missing payment request error")
	}
	if rentalRepo.updateCalled {
		t.Fatal("repository update should not be called")
	}
}

func TestRentalUpdateStatusBlocksPaidCancellationWithoutRefund(t *testing.T) {
	rentalRepo := &rentalServiceRentalRepo{rental: &models.Rental{
		ID:     44,
		UserID: 7,
		Status: models.RentalStatusConfirmed,
		Payment: &models.Payment{
			Status: models.PaymentStatusPaid,
		},
	}}
	service := NewRentalService(rentalRepo, &rentalServiceCarRepo{})

	err := service.UpdateStatus(context.Background(), 44, models.RentalStatusCancelled)
	if err == nil {
		t.Fatal("expected refund policy error")
	}
	if rentalRepo.updateCalled {
		t.Fatal("repository update should not be called")
	}
}

func TestRentalUpdateStatusAllowsActiveWithPaidPayment(t *testing.T) {
	rentalRepo := &rentalServiceRentalRepo{rental: &models.Rental{
		ID:     44,
		UserID: 7,
		Status: models.RentalStatusConfirmed,
		Payment: &models.Payment{
			Status: models.PaymentStatusPaid,
		},
	}}
	service := NewRentalService(rentalRepo, &rentalServiceCarRepo{})

	if err := service.UpdateStatus(context.Background(), 44, models.RentalStatusActive); err != nil {
		t.Fatalf("activate paid confirmed rental: %v", err)
	}
	if !rentalRepo.updateCalled || rentalRepo.updatedID != 44 || rentalRepo.updatedStatus != models.RentalStatusActive {
		t.Fatalf("expected rental 44 to be activated, got id=%d status=%s", rentalRepo.updatedID, rentalRepo.updatedStatus)
	}
}

func TestRentalUpdateStatusBlocksInvalidActiveRentalResetToApproved(t *testing.T) {
	rentalRepo := &rentalServiceRentalRepo{rental: &models.Rental{
		ID:     44,
		UserID: 7,
		Status: models.RentalStatusActive,
	}}
	service := NewRentalService(rentalRepo, &rentalServiceCarRepo{})

	if err := service.UpdateStatus(context.Background(), 44, models.RentalStatusApproved); err == nil {
		t.Fatal("expected invalid transition error")
	}
	if rentalRepo.updateCalled {
		t.Fatal("repository update should not be called")
	}
}

func TestRentalCancelRejectsConfirmedRentalOnPickupDate(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	rentalRepo := &rentalServiceRentalRepo{rental: &models.Rental{
		ID:        44,
		UserID:    7,
		StartDate: today,
		Status:    models.RentalStatusConfirmed,
	}}
	service := NewRentalService(rentalRepo, &rentalServiceCarRepo{})

	err := service.Cancel(context.Background(), 7, models.RoleCustomer, 44)
	if err == nil {
		t.Fatal("expected cancellation policy error")
	}
	if rentalRepo.cancelCalled {
		t.Fatal("repository cancel should not be called")
	}
}

func TestRentalCancelAllowsFutureConfirmedRental(t *testing.T) {
	tomorrow := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	rentalRepo := &rentalServiceRentalRepo{rental: &models.Rental{
		ID:        44,
		UserID:    7,
		StartDate: tomorrow,
		Status:    models.RentalStatusConfirmed,
	}}
	service := NewRentalService(rentalRepo, &rentalServiceCarRepo{})

	if err := service.Cancel(context.Background(), 7, models.RoleCustomer, 44); err != nil {
		t.Fatalf("cancel future confirmed rental: %v", err)
	}
	if !rentalRepo.cancelCalled || rentalRepo.cancelledID != 44 {
		t.Fatalf("expected rental 44 to be cancelled, got %d", rentalRepo.cancelledID)
	}
}
