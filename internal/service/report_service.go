package service

import (
	"context"
	"time"

	"car-rental-system/internal/repository"
	"car-rental-system/pkg/apperror"
)

type ReportService struct {
	reports repository.ReportRepository
}

type ReportInput struct {
	StartDate string
	EndDate   string
}

func NewReportService(reports repository.ReportRepository) *ReportService {
	return &ReportService{reports: reports}
}

func (s *ReportService) Summary(ctx context.Context, input ReportInput) (*repository.ReportSummary, error) {
	end := time.Now().UTC()
	start := end.AddDate(0, -1, 0)

	var err error
	if input.StartDate != "" {
		start, err = time.Parse(time.DateOnly, input.StartDate)
		if err != nil {
			return nil, apperror.ErrInvalidDate
		}
	}
	if input.EndDate != "" {
		end, err = time.Parse(time.DateOnly, input.EndDate)
		if err != nil {
			return nil, apperror.ErrInvalidDate
		}
	}
	if end.Before(start) {
		return nil, apperror.ErrInvalidDateSpan
	}

	return s.reports.Summary(ctx, start, end)
}
