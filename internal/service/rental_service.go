package service

import (
	"context"
	"sort"
	"time"

	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
	"car-rental-system/pkg/apperror"
)

type RentalService struct {
	rentals     repository.RentalRepository
	cars        repository.CarRepository
	maintenance repository.MaintenanceRepository
}

type RentalInput struct {
	CarID     int64  `json:"car_id" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

type RentalStatusInput struct {
	Status models.RentalStatus `json:"status" binding:"required,oneof=requested approved rejected pending_payment confirmed active cancelled completed"`
}

type RentalListInput struct {
	Status        models.RentalStatus
	PaymentStatus models.PaymentStatus
	UserID        int64
	CarID         int64
	StartFrom     string
	EndTo         string
	Page          int
	PageSize      int
}

type AvailabilityInput struct {
	CarID     int64
	StartDate string
	EndDate   string
}

type AvailabilityResult struct {
	CarID       int64   `json:"car_id"`
	Available   bool    `json:"available"`
	Reason      string  `json:"reason,omitempty"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	Days        int     `json:"days"`
	DailyRate   float64 `json:"daily_rate"`
	TotalAmount float64 `json:"total_amount"`
}

type AvailabilityCalendarInput struct {
	CarID int64
	Month string
}

type AvailabilityCalendarResult struct {
	CarID       int64                          `json:"car_id"`
	Month       string                         `json:"month"`
	BlockedDays []string                       `json:"blocked_days"`
	Ranges      []repository.AvailabilityRange `json:"ranges"`
}

func NewRentalService(rentals repository.RentalRepository, cars repository.CarRepository, maintenance ...repository.MaintenanceRepository) *RentalService {
	var maintenanceRepo repository.MaintenanceRepository
	if len(maintenance) > 0 {
		maintenanceRepo = maintenance[0]
	}
	return &RentalService{rentals: rentals, cars: cars, maintenance: maintenanceRepo}
}

func (s *RentalService) Create(ctx context.Context, userID int64, input RentalInput) (*models.Rental, error) {
	startDate, endDate, days, err := parseRentalDates(input.StartDate, input.EndDate)
	if err != nil {
		return nil, err
	}

	car, err := s.cars.FindByID(ctx, input.CarID)
	if err != nil {
		return nil, err
	}

	if car.Status == models.CarStatusMaintenance || car.Status == models.CarStatusInactive {
		return nil, apperror.ErrCarUnavailable
	}

	available, err := s.cars.IsAvailable(ctx, input.CarID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, apperror.ErrDoubleBooking
	}

	total := float64(days) * car.DailyRate

	rental := &models.Rental{
		UserID:      userID,
		CarID:       input.CarID,
		StartDate:   startDate,
		EndDate:     endDate,
		TotalAmount: total,
		Status:      models.RentalStatusRequested,
	}

	if err := s.rentals.CreateWithAvailability(ctx, rental); err != nil {
		return nil, err
	}

	return rental, nil
}

func (s *RentalService) CheckAvailability(ctx context.Context, input AvailabilityInput) (*AvailabilityResult, error) {
	startDate, endDate, days, err := parseRentalDates(input.StartDate, input.EndDate)
	if err != nil {
		return nil, err
	}

	car, err := s.cars.FindByID(ctx, input.CarID)
	if err != nil {
		return nil, err
	}

	result := &AvailabilityResult{
		CarID:       input.CarID,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
		Days:        days,
		DailyRate:   car.DailyRate,
		TotalAmount: float64(days) * car.DailyRate,
	}

	if car.Status == models.CarStatusMaintenance || car.Status == models.CarStatusInactive {
		result.Available = false
		result.Reason = "car is not available for rent"
		return result, nil
	}

	available, err := s.cars.IsAvailable(ctx, input.CarID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	result.Available = available
	if !available {
		result.Reason = "car is already booked for selected dates"
	}

	return result, nil
}

func (s *RentalService) AvailabilityCalendar(ctx context.Context, input AvailabilityCalendarInput) (*AvailabilityCalendarResult, error) {
	month := input.Month
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	monthStart, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, apperror.ErrInvalidDate
	}
	monthEnd := monthStart.AddDate(0, 1, -1)

	if _, err := s.cars.FindByID(ctx, input.CarID); err != nil {
		return nil, err
	}

	ranges, err := s.rentals.ListCalendarRanges(ctx, input.CarID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	if s.maintenance != nil {
		maintenanceRanges, err := s.maintenance.ListCalendarRanges(ctx, input.CarID, monthStart, monthEnd)
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, maintenanceRanges...)
	}

	blocked := make(map[string]struct{})
	for _, item := range ranges {
		start := maxTime(item.StartDate, monthStart)
		end := minTime(item.EndDate, monthEnd)
		for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
			blocked[day.Format(time.DateOnly)] = struct{}{}
		}
	}

	days := make([]string, 0, len(blocked))
	for day := range blocked {
		days = append(days, day)
	}
	sort.Strings(days)

	return &AvailabilityCalendarResult{CarID: input.CarID, Month: month, BlockedDays: days, Ranges: ranges}, nil
}

func parseRentalDates(start, end string) (time.Time, time.Time, int, error) {
	startDate, err := time.Parse(time.DateOnly, start)
	if err != nil {
		return time.Time{}, time.Time{}, 0, apperror.Wrap(apperror.ErrInvalidDate, err)
	}

	endDate, err := time.Parse(time.DateOnly, end)
	if err != nil {
		return time.Time{}, time.Time{}, 0, apperror.Wrap(apperror.ErrInvalidDate, err)
	}

	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, 0, apperror.ErrInvalidDateSpan
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if startDate.Before(today) {
		return time.Time{}, time.Time{}, 0, apperror.New(400, "start_date cannot be in the past")
	}

	days := int(endDate.Sub(startDate).Hours()/24) + 1
	return startDate, endDate, days, nil
}

func parseOptionalDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	return time.Parse(time.DateOnly, value)
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (s *RentalService) ListAll(ctx context.Context, input RentalListInput) (*repository.RentalListResult, error) {
	if input.Status != "" &&
		input.Status != models.RentalStatusPendingPayment &&
		input.Status != models.RentalStatusRequested &&
		input.Status != models.RentalStatusApproved &&
		input.Status != models.RentalStatusRejected &&
		input.Status != models.RentalStatusConfirmed &&
		input.Status != models.RentalStatusActive &&
		input.Status != models.RentalStatusCancelled &&
		input.Status != models.RentalStatusCompleted {
		return nil, apperror.New(400, "invalid rental status")
	}
	if input.PaymentStatus != "" &&
		input.PaymentStatus != models.PaymentStatusPending &&
		input.PaymentStatus != models.PaymentStatusPaid &&
		input.PaymentStatus != models.PaymentStatusFailed &&
		input.PaymentStatus != models.PaymentStatusRefunded {
		return nil, apperror.New(400, "invalid payment status")
	}

	startFrom, err := parseOptionalDate(input.StartFrom)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInvalidDate, err)
	}
	endTo, err := parseOptionalDate(input.EndTo)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInvalidDate, err)
	}
	if !startFrom.IsZero() && !endTo.IsZero() && endTo.Before(startFrom) {
		return nil, apperror.ErrInvalidDateSpan
	}

	return s.rentals.ListAll(ctx, repository.RentalListFilter{
		Status:        input.Status,
		PaymentStatus: input.PaymentStatus,
		UserID:        input.UserID,
		CarID:         input.CarID,
		StartFrom:     startFrom,
		EndTo:         endTo,
		Page:          input.Page,
		PageSize:      input.PageSize,
	})
}

func (s *RentalService) ListByUserID(ctx context.Context, userID int64) ([]models.Rental, error) {
	return s.rentals.ListByUserID(ctx, userID)
}

func (s *RentalService) SyncLifecycle(ctx context.Context, now time.Time) (repository.RentalLifecycleSyncResult, error) {
	return s.rentals.SyncLifecycle(ctx, now)
}

func (s *RentalService) FindByID(ctx context.Context, userID int64, role models.UserRole, id int64) (*models.Rental, error) {
	rental, err := s.rentals.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rental.UserID != userID && role != models.RoleAdmin && role != models.RoleSuperAdmin {
		return nil, apperror.ErrForbidden
	}

	return rental, nil
}

func (s *RentalService) UpdateStatus(ctx context.Context, id int64, status models.RentalStatus) error {
	rental, err := s.rentals.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := validateRentalStatusChange(rental, status); err != nil {
		return err
	}

	return s.rentals.UpdateStatus(ctx, id, status)
}

func validateRentalStatusChange(rental *models.Rental, next models.RentalStatus) error {
	if next == rental.Status {
		return nil
	}

	switch next {
	case models.RentalStatusRequested:
		return apperror.New(400, "rental cannot be moved back to requested")
	case models.RentalStatusApproved:
		if rental.Status != models.RentalStatusRequested &&
			rental.Status != models.RentalStatusRejected &&
			rental.Status != models.RentalStatusCancelled {
			return apperror.New(400, "only requested, rejected, or cancelled rentals can be approved")
		}
		return ensureRentalHasNoOpenPayment(rental, "approving")
	case models.RentalStatusRejected:
		if rental.Status != models.RentalStatusRequested && rental.Status != models.RentalStatusApproved {
			return apperror.New(400, "only requested or approved rentals can be rejected")
		}
		return ensureRentalHasNoOpenPayment(rental, "rejecting")
	case models.RentalStatusCancelled:
		if rental.Status == models.RentalStatusCompleted {
			return apperror.New(400, "completed rental cannot be cancelled")
		}
		return ensureRentalHasNoOpenPayment(rental, "cancelling")
	case models.RentalStatusPendingPayment:
		return apperror.New(400, "create a payment request to mark rental as pending payment")
	case models.RentalStatusConfirmed:
		return apperror.New(400, "confirm the payment to confirm rental")
	case models.RentalStatusActive:
		if !rentalPaymentPaid(rental) {
			return apperror.New(400, "payment must be paid before activating rental")
		}
		if rental.Status != models.RentalStatusConfirmed {
			return apperror.New(400, "only confirmed rentals can be activated")
		}
		return nil
	case models.RentalStatusCompleted:
		if !rentalPaymentPaid(rental) {
			return apperror.New(400, "payment must be paid before completing rental")
		}
		if rental.Status != models.RentalStatusActive {
			return apperror.New(400, "only active rentals can be completed")
		}
		return nil
	default:
		return apperror.New(400, "invalid rental status")
	}
}

func rentalPaymentPaid(rental *models.Rental) bool {
	return rental.Payment != nil && rental.Payment.Status == models.PaymentStatusPaid
}

func ensureRentalHasNoOpenPayment(rental *models.Rental, action string) error {
	if rental.Payment == nil {
		return nil
	}

	switch rental.Payment.Status {
	case models.PaymentStatusFailed, models.PaymentStatusRefunded:
		return nil
	case models.PaymentStatusPending:
		return apperror.New(400, "fail the pending payment before "+action+" rental")
	case models.PaymentStatusPaid:
		return apperror.New(400, "refund the paid payment before "+action+" rental")
	default:
		return apperror.New(400, "payment must be resolved before "+action+" rental")
	}
}

func (s *RentalService) Approve(ctx context.Context, id int64) (*models.Rental, error) {
	rental, err := s.rentals.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rental.Status != models.RentalStatusRequested {
		return nil, apperror.New(400, "only requested rentals can be approved")
	}
	if err := ensureRentalHasNoOpenPayment(rental, "approving"); err != nil {
		return nil, err
	}
	if err := s.rentals.UpdateStatus(ctx, id, models.RentalStatusApproved); err != nil {
		return nil, err
	}
	rental.Status = models.RentalStatusApproved
	return rental, nil
}

func (s *RentalService) Reject(ctx context.Context, id int64) (*models.Rental, error) {
	rental, err := s.rentals.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rental.Status != models.RentalStatusRequested && rental.Status != models.RentalStatusApproved {
		return nil, apperror.New(400, "only requested or approved rentals can be rejected")
	}
	if err := ensureRentalHasNoOpenPayment(rental, "rejecting"); err != nil {
		return nil, err
	}
	if err := s.rentals.UpdateStatus(ctx, id, models.RentalStatusRejected); err != nil {
		return nil, err
	}
	rental.Status = models.RentalStatusRejected
	return rental, nil
}

func (s *RentalService) Cancel(ctx context.Context, userID int64, role models.UserRole, id int64) error {
	rental, err := s.rentals.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if rental.UserID != userID && role != models.RoleAdmin && role != models.RoleSuperAdmin {
		return apperror.ErrForbidden
	}
	if rental.Status == models.RentalStatusCancelled {
		return apperror.New(400, "rental is already cancelled")
	}
	if rental.Status == models.RentalStatusCompleted {
		return apperror.New(400, "completed rental cannot be cancelled")
	}
	if err := ensureRentalHasNoOpenPayment(rental, "cancelling"); err != nil {
		return err
	}
	if rental.Status == models.RentalStatusConfirmed {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		if !rental.StartDate.After(today) {
			return apperror.New(400, "confirmed rental can be cancelled only before pickup date")
		}
	}

	return s.rentals.Cancel(ctx, id)
}
