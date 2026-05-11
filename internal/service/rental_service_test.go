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
	rental       *models.Rental
	created      *models.Rental
	cancelledID  int64
	cancelCalled bool
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
func (r *rentalServiceRentalRepo) UpdateStatus(context.Context, int64, models.RentalStatus) error {
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

	_, err := service.Create(context.Background(), 7, RentalInput{
		CarID:     10,
		StartDate: "2026-05-10",
		EndDate:   "2026-05-12",
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

	rental, err := service.Create(context.Background(), 7, RentalInput{
		CarID:     10,
		StartDate: "2026-05-10",
		EndDate:   "2026-05-12",
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
