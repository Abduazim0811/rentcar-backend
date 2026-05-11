package handler

import (
	"car-rental-system/internal/service"
	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	reports *service.ReportService
}

func NewReportHandler(reports *service.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

func (h *ReportHandler) Summary(c *gin.Context) {
	summary, err := h.reports.Summary(c.Request.Context(), service.ReportInput{
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, summary)
}
