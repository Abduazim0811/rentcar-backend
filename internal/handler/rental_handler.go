package handler

import (
	"context"
	"net/http"

	"car-rental-system/internal/models"
	"car-rental-system/internal/service"
	"car-rental-system/pkg/apperror"
	"car-rental-system/pkg/response"
	"car-rental-system/pkg/validator"

	"github.com/gin-gonic/gin"
)

type RentalHandler struct {
	rentals       *service.RentalService
	notifications *service.NotificationService
	emails        service.EmailSender
	audit         *service.AuditService
}

func NewRentalHandler(rentals *service.RentalService, extras ...any) *RentalHandler {
	h := &RentalHandler{rentals: rentals}
	for _, extra := range extras {
		switch value := extra.(type) {
		case *service.NotificationService:
			h.notifications = value
		case service.EmailSender:
			h.emails = value
		case *service.AuditService:
			h.audit = value
		}
	}
	return h
}

func (h *RentalHandler) Availability(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	if startDate == "" || endDate == "" {
		response.Error(c, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	availability, err := h.rentals.CheckAvailability(c.Request.Context(), service.AvailabilityInput{
		CarID:     id,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, availability)
}

func (h *RentalHandler) Calendar(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	result, err := h.rentals.AvailabilityCalendar(c.Request.Context(), service.AvailabilityCalendarInput{
		CarID: id,
		Month: c.Query("month"),
	})
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *RentalHandler) Create(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}

	var input service.RentalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	rental, err := h.rentals.Create(c.Request.Context(), userID.(int64), input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	if h.notifications != nil {
		userIDValue := userID.(int64)
		_ = h.notifications.Create(c.Request.Context(), service.NotificationInput{UserID: &userIDValue, Title: "Rental requested", Message: "Your booking request was sent to admin for approval.", Type: models.NotificationTypeInfo})
	}
	response.Created(c, rental)
}

func (h *RentalHandler) MyRentals(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}

	rentals, err := h.rentals.ListByUserID(c.Request.Context(), userID.(int64))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, rentals)
}

func (h *RentalHandler) GetByID(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}

	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	rental, err := h.rentals.FindByID(c.Request.Context(), userID.(int64), roleFromContext(c), id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, rental)
}

func (h *RentalHandler) ListAll(c *gin.Context) {
	input, err := parseRentalListInput(c)
	if err != nil {
		response.FromError(c, err)
		return
	}

	rentals, err := h.rentals.ListAll(c.Request.Context(), input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, rentals)
}

func (h *RentalHandler) UpdateStatus(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	var input service.RentalStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	rental, err := h.rentals.FindByID(c.Request.Context(), 0, models.RoleSuperAdmin, id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	before := *rental
	previousStatus := rental.Status

	if err := h.rentals.UpdateStatus(c.Request.Context(), id, input.Status); err != nil {
		response.FromError(c, err)
		return
	}
	if previousStatus != input.Status && shouldNotifyRentalStatus(input.Status) {
		rental.Status = input.Status
		h.notifyRentalStatus(c.Request.Context(), rental)
	}

	after := before
	after.Status = input.Status
	audit(c, h.audit, "rental.status_updated", "rental", id, auditMetadata(&before, &after))
	response.OK(c, gin.H{"updated": true})
}

func (h *RentalHandler) Approve(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	before, err := h.rentals.FindByID(c.Request.Context(), 0, models.RoleSuperAdmin, id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	rental, err := h.rentals.Approve(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	h.notifyRentalStatus(c.Request.Context(), rental)
	audit(c, h.audit, "rental.approved", "rental", id, auditMetadata(before, rental))
	response.OK(c, rental)
}

func (h *RentalHandler) Reject(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	before, err := h.rentals.FindByID(c.Request.Context(), 0, models.RoleSuperAdmin, id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	rental, err := h.rentals.Reject(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	h.notifyRentalStatus(c.Request.Context(), rental)
	audit(c, h.audit, "rental.rejected", "rental", id, auditMetadata(before, rental))
	response.OK(c, rental)
}

func (h *RentalHandler) Cancel(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}

	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	role := roleFromContext(c)
	before, err := h.rentals.FindByID(c.Request.Context(), 0, models.RoleSuperAdmin, id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	if err := h.rentals.Cancel(c.Request.Context(), userID.(int64), role, id); err != nil {
		response.FromError(c, err)
		return
	}
	var after *models.Rental
	if role == models.RoleAdmin || role == models.RoleSuperAdmin {
		if rental, err := h.rentals.FindByID(c.Request.Context(), 0, models.RoleSuperAdmin, id); err == nil {
			after = rental
			h.notifyRentalStatus(c.Request.Context(), rental)
		}
	}
	if after == nil {
		copy := *before
		copy.Status = models.RentalStatusCancelled
		after = &copy
	}

	audit(c, h.audit, "rental.cancelled", "rental", id, auditMetadata(before, after))
	response.OK(c, gin.H{"cancelled": true})
}

func (h *RentalHandler) notifyRentalStatus(ctx context.Context, rental *models.Rental) {
	if rental == nil {
		return
	}

	title, message, notificationType, ok := rentalStatusNotification(rental.Status)
	if !ok {
		return
	}

	if h.notifications != nil {
		_ = h.notifications.Create(ctx, service.NotificationInput{
			UserID:  &rental.UserID,
			Title:   title,
			Message: message,
			Type:    notificationType,
		})
	}
	if h.emails != nil && rental.User != nil && rental.User.Email != "" {
		_ = h.emails.SendRentalStatusUpdate(ctx, rental.User.Email, rental.User.Name, rental)
	}
}

func shouldNotifyRentalStatus(status models.RentalStatus) bool {
	_, _, _, ok := rentalStatusNotification(status)
	return ok
}

func rentalStatusNotification(status models.RentalStatus) (string, string, models.NotificationType, bool) {
	switch status {
	case models.RentalStatusApproved:
		return "Rental approved", "Your booking was approved. You can create a payment request now.", models.NotificationTypeSuccess, true
	case models.RentalStatusRejected:
		return "Rental rejected", "Your booking request was rejected by admin.", models.NotificationTypeWarning, true
	case models.RentalStatusCancelled:
		return "Rental cancelled", "Your booking was cancelled by admin.", models.NotificationTypeWarning, true
	default:
		return "", "", "", false
	}
}

func roleFromContext(c *gin.Context) models.UserRole {
	role, _ := c.Get("role")
	value, _ := role.(string)
	return models.UserRole(value)
}

func parseRentalListInput(c *gin.Context) (service.RentalListInput, error) {
	userID, err := parseOptionalID(c.Query("user_id"))
	if err != nil {
		return service.RentalListInput{}, apperror.New(http.StatusBadRequest, "invalid user_id")
	}
	carID, err := parseOptionalID(c.Query("car_id"))
	if err != nil {
		return service.RentalListInput{}, apperror.New(http.StatusBadRequest, "invalid car_id")
	}
	page, err := parseOptionalInt(c.Query("page"))
	if err != nil {
		return service.RentalListInput{}, apperror.New(http.StatusBadRequest, "invalid page")
	}
	pageSize, err := parseOptionalInt(c.Query("page_size"))
	if err != nil {
		return service.RentalListInput{}, apperror.New(http.StatusBadRequest, "invalid page_size")
	}

	return service.RentalListInput{
		Status:        models.RentalStatus(c.Query("status")),
		PaymentStatus: models.PaymentStatus(c.Query("payment_status")),
		UserID:        userID,
		CarID:         carID,
		StartFrom:     c.Query("start_from"),
		EndTo:         c.Query("end_to"),
		Page:          page,
		PageSize:      pageSize,
	}, nil
}

func parseOptionalID(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	return parseID(value)
}
