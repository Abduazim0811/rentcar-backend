package handler

import (
	"net/http"

	"car-rental-system/internal/repository"
	"car-rental-system/internal/service"
	"car-rental-system/pkg/apperror"
	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	audits *service.AuditService
}

func NewAuditHandler(audits *service.AuditService) *AuditHandler {
	return &AuditHandler{audits: audits}
}

func (h *AuditHandler) List(c *gin.Context) {
	actorID, err := parseOptionalID(c.Query("actor_id"))
	if err != nil {
		response.FromError(c, apperror.New(http.StatusBadRequest, "invalid actor_id"))
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

	items, err := h.audits.List(c.Request.Context(), repository.AuditListFilter{
		ActorID:    actorID,
		EntityType: c.Query("entity_type"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, items)
}
