package handler

import (
	"net/http"
	"strconv"

	"car-rental-system/internal/models"
	"car-rental-system/internal/service"
	"car-rental-system/pkg/apperror"
	"car-rental-system/pkg/response"
	"car-rental-system/pkg/validator"

	"github.com/gin-gonic/gin"
)

type CarHandler struct {
	cars  *service.CarService
	audit *service.AuditService
}

func NewCarHandler(cars *service.CarService, audits ...*service.AuditService) *CarHandler {
	var auditService *service.AuditService
	if len(audits) > 0 {
		auditService = audits[0]
	}
	return &CarHandler{cars: cars, audit: auditService}
}

func (h *CarHandler) Create(c *gin.Context) {
	var input service.CarInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	car, err := h.cars.Create(c.Request.Context(), input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	audit(c, h.audit, "car.created", "car", car.ID, auditMetadata(nil, car))
	response.Created(c, car)
}

func (h *CarHandler) Update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	before, err := h.cars.FindByID(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	var input service.CarInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	car, err := h.cars.Update(c.Request.Context(), id, input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	audit(c, h.audit, "car.updated", "car", car.ID, auditMetadata(before, car))
	response.OK(c, car)
}

func (h *CarHandler) Delete(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	before, err := h.cars.FindByID(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	if err := h.cars.Delete(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}

	audit(c, h.audit, "car.deleted", "car", id, auditMetadata(before, nil))
	response.OK(c, gin.H{"deleted": true})
}

func (h *CarHandler) List(c *gin.Context) {
	input, err := parseCarListInput(c)
	if err != nil {
		response.FromError(c, err)
		return
	}
	input.Status = models.CarStatusAvailable

	cars, err := h.cars.List(c.Request.Context(), input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, cars)
}

func (h *CarHandler) ListAdmin(c *gin.Context) {
	input, err := parseCarListInput(c)
	if err != nil {
		response.FromError(c, err)
		return
	}

	cars, err := h.cars.List(c.Request.Context(), input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, cars)
}

func (h *CarHandler) GetByID(c *gin.Context) {
	car, ok := h.findCarByID(c)
	if !ok {
		return
	}
	if car.Status != models.CarStatusAvailable {
		response.FromError(c, apperror.ErrNotFound)
		return
	}

	response.OK(c, car)
}

func (h *CarHandler) GetAdminByID(c *gin.Context) {
	car, ok := h.findCarByID(c)
	if !ok {
		return
	}

	response.OK(c, car)
}

func (h *CarHandler) findCarByID(c *gin.Context) (*models.Car, bool) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return nil, false
	}

	car, err := h.cars.FindByID(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return nil, false
	}

	return car, true
}

func parseID(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

func parseCarListInput(c *gin.Context) (service.CarListInput, error) {
	minYear, err := parseOptionalInt(c.Query("min_year"))
	if err != nil {
		return service.CarListInput{}, apperror.New(http.StatusBadRequest, "invalid min_year")
	}
	maxYear, err := parseOptionalInt(c.Query("max_year"))
	if err != nil {
		return service.CarListInput{}, apperror.New(http.StatusBadRequest, "invalid max_year")
	}
	minRate, err := parseOptionalFloat(c.Query("min_rate"))
	if err != nil {
		return service.CarListInput{}, apperror.New(http.StatusBadRequest, "invalid min_rate")
	}
	maxRate, err := parseOptionalFloat(c.Query("max_rate"))
	if err != nil {
		return service.CarListInput{}, apperror.New(http.StatusBadRequest, "invalid max_rate")
	}
	page, err := parseOptionalInt(c.Query("page"))
	if err != nil {
		return service.CarListInput{}, apperror.New(http.StatusBadRequest, "invalid page")
	}
	pageSize, err := parseOptionalInt(c.Query("page_size"))
	if err != nil {
		return service.CarListInput{}, apperror.New(http.StatusBadRequest, "invalid page_size")
	}

	return service.CarListInput{
		Search:       c.Query("q"),
		Status:       models.CarStatus(c.Query("status")),
		MinYear:      minYear,
		MaxYear:      maxYear,
		MinDailyRate: minRate,
		MaxDailyRate: maxRate,
		AvailableOn:  c.Query("available_on"),
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

func parseOptionalInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func parseOptionalFloat(value string) (float64, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.ParseFloat(value, 64)
}
