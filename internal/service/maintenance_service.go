package service

import (
	"context"

	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
	"car-rental-system/pkg/apperror"
)

type MaintenanceService struct {
	maintenance repository.MaintenanceRepository
	cars        repository.CarRepository
}

type MaintenanceInput struct {
	CarID     int64                    `json:"car_id" binding:"required"`
	StartDate string                   `json:"start_date" binding:"required"`
	EndDate   string                   `json:"end_date" binding:"required"`
	Reason    string                   `json:"reason" binding:"required"`
	Status    models.MaintenanceStatus `json:"status" binding:"omitempty,oneof=scheduled in_progress completed cancelled"`
	Notes     string                   `json:"notes"`
}

type MaintenanceListInput struct {
	CarID    int64
	Status   models.MaintenanceStatus
	Page     int
	PageSize int
}

func NewMaintenanceService(maintenance repository.MaintenanceRepository, cars repository.CarRepository) *MaintenanceService {
	return &MaintenanceService{maintenance: maintenance, cars: cars}
}

func (s *MaintenanceService) Create(ctx context.Context, actorID int64, input MaintenanceInput) (*models.CarMaintenance, error) {
	startDate, endDate, _, err := parseRentalDates(input.StartDate, input.EndDate)
	if err != nil {
		return nil, err
	}
	if _, err := s.cars.FindByID(ctx, input.CarID); err != nil {
		return nil, err
	}
	if input.Status == "" {
		input.Status = models.MaintenanceStatusScheduled
	}
	if !validMaintenanceStatus(input.Status) {
		return nil, apperror.New(400, "invalid maintenance status")
	}

	item := &models.CarMaintenance{
		CarID:     input.CarID,
		StartDate: startDate,
		EndDate:   endDate,
		Reason:    input.Reason,
		Status:    input.Status,
		Notes:     input.Notes,
		CreatedBy: &actorID,
	}
	if err := s.maintenance.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MaintenanceService) Update(ctx context.Context, id int64, input MaintenanceInput) (*models.CarMaintenance, error) {
	startDate, endDate, _, err := parseRentalDates(input.StartDate, input.EndDate)
	if err != nil {
		return nil, err
	}
	if _, err := s.cars.FindByID(ctx, input.CarID); err != nil {
		return nil, err
	}
	if input.Status == "" {
		input.Status = models.MaintenanceStatusScheduled
	}
	if !validMaintenanceStatus(input.Status) {
		return nil, apperror.New(400, "invalid maintenance status")
	}

	item := &models.CarMaintenance{ID: id, CarID: input.CarID, StartDate: startDate, EndDate: endDate, Reason: input.Reason, Status: input.Status, Notes: input.Notes}
	if err := s.maintenance.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *MaintenanceService) Delete(ctx context.Context, id int64) error {
	return s.maintenance.Delete(ctx, id)
}

func (s *MaintenanceService) List(ctx context.Context, input MaintenanceListInput) (*repository.MaintenanceListResult, error) {
	if input.Status != "" && !validMaintenanceStatus(input.Status) {
		return nil, apperror.New(400, "invalid maintenance status")
	}
	return s.maintenance.List(ctx, repository.MaintenanceListFilter{CarID: input.CarID, Status: input.Status, Page: input.Page, PageSize: input.PageSize})
}

func validMaintenanceStatus(status models.MaintenanceStatus) bool {
	return status == models.MaintenanceStatusScheduled ||
		status == models.MaintenanceStatusInProgress ||
		status == models.MaintenanceStatusCompleted ||
		status == models.MaintenanceStatusCancelled
}
