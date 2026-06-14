package service

import (
	"context"
	"testing"
	"time"

	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
	"car-rental-system/pkg/apperror"
)

type maintenanceServiceCarRepo struct {
	cars    map[int64]*models.Car
	updates []models.CarStatus
}

func (r *maintenanceServiceCarRepo) Create(context.Context, *models.Car) error { return nil }
func (r *maintenanceServiceCarRepo) Delete(context.Context, int64) error       { return nil }
func (r *maintenanceServiceCarRepo) IsAvailable(context.Context, int64, time.Time, time.Time) (bool, error) {
	return true, nil
}
func (r *maintenanceServiceCarRepo) List(context.Context, repository.CarListFilter) (*repository.CarListResult, error) {
	return nil, nil
}
func (r *maintenanceServiceCarRepo) FindByID(_ context.Context, id int64) (*models.Car, error) {
	car, ok := r.cars[id]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	copy := *car
	return &copy, nil
}
func (r *maintenanceServiceCarRepo) Update(_ context.Context, car *models.Car) error {
	copy := *car
	r.cars[car.ID] = &copy
	r.updates = append(r.updates, car.Status)
	return nil
}

type maintenanceServiceMaintenanceRepo struct {
	nextID int64
	items  map[int64]*models.CarMaintenance
}

func (r *maintenanceServiceMaintenanceRepo) Create(_ context.Context, item *models.CarMaintenance) error {
	if r.nextID == 0 {
		r.nextID = 1
	}
	item.ID = r.nextID
	r.nextID++
	copy := *item
	r.items[item.ID] = &copy
	return nil
}
func (r *maintenanceServiceMaintenanceRepo) Update(_ context.Context, item *models.CarMaintenance) error {
	if _, ok := r.items[item.ID]; !ok {
		return apperror.ErrNotFound
	}
	copy := *item
	r.items[item.ID] = &copy
	return nil
}
func (r *maintenanceServiceMaintenanceRepo) Delete(_ context.Context, id int64) error {
	if _, ok := r.items[id]; !ok {
		return apperror.ErrNotFound
	}
	delete(r.items, id)
	return nil
}
func (r *maintenanceServiceMaintenanceRepo) FindByID(_ context.Context, id int64) (*models.CarMaintenance, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	copy := *item
	return &copy, nil
}
func (r *maintenanceServiceMaintenanceRepo) List(_ context.Context, filter repository.MaintenanceListFilter) (*repository.MaintenanceListResult, error) {
	var items []models.CarMaintenance
	for _, item := range r.items {
		if filter.CarID > 0 && item.CarID != filter.CarID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, *item)
	}
	return &repository.MaintenanceListResult{Items: items, Total: int64(len(items)), Page: 1, PageSize: 1, TotalPages: len(items)}, nil
}
func (r *maintenanceServiceMaintenanceRepo) ListCalendarRanges(context.Context, int64, time.Time, time.Time) ([]repository.AvailabilityRange, error) {
	return nil, nil
}

func TestMaintenanceCreateSetsCarStatusToMaintenance(t *testing.T) {
	carRepo := &maintenanceServiceCarRepo{
		cars: map[int64]*models.Car{
			10: {ID: 10, Status: models.CarStatusAvailable},
		},
	}
	maintenanceRepo := &maintenanceServiceMaintenanceRepo{items: map[int64]*models.CarMaintenance{}}
	service := NewMaintenanceService(maintenanceRepo, carRepo)

	_, err := service.Create(context.Background(), 1, MaintenanceInput{
		CarID:     10,
		StartDate: testFutureDate(1),
		EndDate:   testFutureDate(2),
		Reason:    "Oil service",
		Status:    models.MaintenanceStatusScheduled,
	})
	if err != nil {
		t.Fatalf("create maintenance: %v", err)
	}

	if got := carRepo.cars[10].Status; got != models.CarStatusMaintenance {
		t.Fatalf("expected car status maintenance, got %s", got)
	}
}

func TestMaintenanceUpdateCompletedRestoresAvailableWhenNoActiveMaintenance(t *testing.T) {
	carRepo := &maintenanceServiceCarRepo{
		cars: map[int64]*models.Car{
			10: {ID: 10, Status: models.CarStatusMaintenance},
		},
	}
	maintenanceRepo := &maintenanceServiceMaintenanceRepo{items: map[int64]*models.CarMaintenance{
		5: {
			ID:        5,
			CarID:     10,
			StartDate: mustParseDate(testFutureDate(1)),
			EndDate:   mustParseDate(testFutureDate(2)),
			Reason:    "Oil service",
			Status:    models.MaintenanceStatusScheduled,
		},
	}}
	service := NewMaintenanceService(maintenanceRepo, carRepo)

	_, err := service.Update(context.Background(), 5, MaintenanceInput{
		CarID:     10,
		StartDate: testFutureDate(1),
		EndDate:   testFutureDate(2),
		Reason:    "Oil service",
		Status:    models.MaintenanceStatusCompleted,
	})
	if err != nil {
		t.Fatalf("update maintenance: %v", err)
	}

	if got := carRepo.cars[10].Status; got != models.CarStatusAvailable {
		t.Fatalf("expected car status available, got %s", got)
	}
}

func testFutureDate(days int) string {
	return time.Now().UTC().AddDate(0, 0, days).Format(time.DateOnly)
}

func mustParseDate(value string) time.Time {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
