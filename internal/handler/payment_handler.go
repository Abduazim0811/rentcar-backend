package handler

import (
	"net/http"

	"car-rental-system/internal/models"
	"car-rental-system/internal/service"
	"car-rental-system/pkg/apperror"
	"car-rental-system/pkg/response"
	"car-rental-system/pkg/validator"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	payments *service.PaymentService
	invoices *service.InvoiceService
	audit    *service.AuditService
}

func NewPaymentHandler(payments *service.PaymentService, extras ...any) *PaymentHandler {
	h := &PaymentHandler{payments: payments}
	for _, extra := range extras {
		switch value := extra.(type) {
		case *service.InvoiceService:
			h.invoices = value
		case *service.AuditService:
			h.audit = value
		}
	}
	return h
}

func (h *PaymentHandler) Create(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}

	roleValue, _ := c.Get("role")
	roleString, _ := roleValue.(string)
	role := models.UserRole(roleString)

	var input service.PaymentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	payment, err := h.payments.Create(c.Request.Context(), userID.(int64), role, input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	audit(c, h.audit, "payment.created", "payment", payment.ID, "{}")
	response.Created(c, payment)
}

func (h *PaymentHandler) Confirm(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	payment, err := h.payments.FindByID(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	if err := h.payments.Confirm(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	if h.invoices != nil {
		_ = h.invoices.MarkPaid(c.Request.Context(), payment.RentalID)
	}

	audit(c, h.audit, "payment.confirmed", "payment", id, "{}")
	response.OK(c, gin.H{"paid": true})
}

func (h *PaymentHandler) Fail(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	if err := h.payments.Fail(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}

	audit(c, h.audit, "payment.failed", "payment", id, "{}")
	response.OK(c, gin.H{"failed": true})
}

func (h *PaymentHandler) Refund(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	if err := h.payments.Refund(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}

	audit(c, h.audit, "payment.refunded", "payment", id, "{}")
	response.OK(c, gin.H{"refunded": true})
}
