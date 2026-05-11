package handler

import (
	"car-rental-system/internal/service"
	"car-rental-system/pkg/apperror"
	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	invoices *service.InvoiceService
}

func NewInvoiceHandler(invoices *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoices: invoices}
}

func (h *InvoiceHandler) GetByRental(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}
	rentalID, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	invoice, err := h.invoices.GetOrCreate(c.Request.Context(), userID.(int64), roleFromContext(c), rentalID)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, invoice)
}
