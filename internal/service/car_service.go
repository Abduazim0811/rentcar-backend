package service

import (
	"context"
	"time"

	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
	"car-rental-system/pkg/apperror"
)

type CarService struct {
	cars repository.CarRepository
}

type CarListInput struct {
	Search       string
	Status       models.CarStatus
	MinYear      int
	MaxYear      int
	MinDailyRate float64
	MaxDailyRate float64
	Page         int
	PageSize     int
}

type CarInput struct {
	Brand       string           `json:"brand" binding:"required"`
	Model       string           `json:"model" binding:"required"`
	Year        int              `json:"year" binding:"required,gte=1900"`
	PlateNumber string           `json:"plate_number" binding:"required"`
	DailyRate   float64          `json:"daily_rate" binding:"required,gt=0"`
	Status      models.CarStatus `json:"status" binding:"omitempty,oneof=available maintenance inactive"`
	Image       string           `json:"image" binding:"omitempty,url"`
}

func NewCarService(cars repository.CarRepository) *CarService {
	return &CarService{cars: cars}
}

func (s *CarService) Create(ctx context.Context, input CarInput) (*models.Car, error) {
	if input.Year > time.Now().Year()+1 {
		return nil, apperror.New(400, "year cannot be in the far future")
	}

	car := &models.Car{
		Brand:       input.Brand,
		Model:       input.Model,
		Year:        input.Year,
		PlateNumber: input.PlateNumber,
		DailyRate:   input.DailyRate,
		Status:      input.Status,
		Image:       input.Image,
	}

	if err := s.cars.Create(ctx, car); err != nil {
		return nil, err
	}

	return car, nil
}

func (s *CarService) Update(ctx context.Context, id int64, input CarInput) (*models.Car, error) {
	if input.Year > time.Now().Year()+1 {
		return nil, apperror.New(400, "year cannot be in the far future")
	}

	car := &models.Car{
		ID:          id,
		Brand:       input.Brand,
		Model:       input.Model,
		Year:        input.Year,
		PlateNumber: input.PlateNumber,
		DailyRate:   input.DailyRate,
		Status:      input.Status,
		Image:       input.Image,
	}

	if car.Status == "" {
		car.Status = models.CarStatusAvailable
	}

	if err := s.cars.Update(ctx, car); err != nil {
		return nil, err
	}

	return car, nil
}

func (s *CarService) Delete(ctx context.Context, id int64) error {
	return s.cars.Delete(ctx, id)
}

func (s *CarService) List(ctx context.Context, input CarListInput) (*repository.CarListResult, error) {
	if input.Status != "" &&
		input.Status != models.CarStatusAvailable &&
		input.Status != models.CarStatusMaintenance &&
		input.Status != models.CarStatusInactive {
		return nil, apperror.New(400, "invalid car status")
	}
	if input.MinYear > 0 && input.MaxYear > 0 && input.MinYear > input.MaxYear {
		return nil, apperror.New(400, "min_year cannot be greater than max_year")
	}
	if input.MinDailyRate > 0 && input.MaxDailyRate > 0 && input.MinDailyRate > input.MaxDailyRate {
		return nil, apperror.New(400, "min_rate cannot be greater than max_rate")
	}

	return s.cars.List(ctx, repository.CarListFilter{
		Search:       input.Search,
		Status:       input.Status,
		MinYear:      input.MinYear,
		MaxYear:      input.MaxYear,
		MinDailyRate: input.MinDailyRate,
		MaxDailyRate: input.MaxDailyRate,
		Page:         input.Page,
		PageSize:     input.PageSize,
	})
}

func (s *CarService) FindByID(ctx context.Context, id int64) (*models.Car, error) {
	return s.cars.FindByID(ctx, id)
}
