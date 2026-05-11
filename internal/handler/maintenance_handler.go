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

type MaintenanceHandler struct {
	maintenance *service.MaintenanceService
	audit       *service.AuditService
}

func NewMaintenanceHandler(maintenance *service.MaintenanceService, audits ...*service.AuditService) *MaintenanceHandler {
	var auditService *service.AuditService
	if len(audits) > 0 {
		auditService = audits[0]
	}
	return &MaintenanceHandler{maintenance: maintenance, audit: auditService}
}

func (h *MaintenanceHandler) Create(c *gin.Context) {
	var input service.MaintenanceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	item, err := h.maintenance.Create(c.Request.Context(), actorIDValue(c), input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	audit(c, h.audit, "maintenance.created", "maintenance", item.ID, "{}")
	response.Created(c, item)
}

func (h *MaintenanceHandler) Update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	var input service.MaintenanceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	item, err := h.maintenance.Update(c.Request.Context(), id, input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	audit(c, h.audit, "maintenance.updated", "maintenance", id, statusMetadata(item.Status))
	response.OK(c, item)
}

func (h *MaintenanceHandler) Delete(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}
	if err := h.maintenance.Delete(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	audit(c, h.audit, "maintenance.deleted", "maintenance", id, "{}")
	response.OK(c, gin.H{"deleted": true})
}

func (h *MaintenanceHandler) List(c *gin.Context) {
	carID, err := parseOptionalID(c.Query("car_id"))
	if err != nil {
		response.FromError(c, apperror.New(http.StatusBadRequest, "invalid car_id"))
		return
	}
	page, err := parseOptionalInt(c.Query("page"))
	if err != nil {
		response.FromError(c, apperror.New(http.StatusBadRequest, "invalid page"))
		return
	}
	pageSize, err := parseOptionalInt(c.Query("page_size"))
	if err != nil {
		response.FromError(c, apperror.New(http.StatusBadRequest, "invalid page_size"))
		return
	}

	items, err := h.maintenance.List(c.Request.Context(), service.MaintenanceListInput{
		CarID:    carID,
		Status:   models.MaintenanceStatus(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, items)
}
