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

type UserHandler struct {
	users *service.UserService
	audit *service.AuditService
}

func NewUserHandler(users *service.UserService, audits ...*service.AuditService) *UserHandler {
	var auditService *service.AuditService
	if len(audits) > 0 {
		auditService = audits[0]
	}
	return &UserHandler{users: users, audit: auditService}
}

func (h *UserHandler) Register(c *gin.Context) {
	var input service.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	result, err := h.users.Register(c.Request.Context(), input, sessionMeta(c))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Created(c, result)
}

func (h *UserHandler) VerifyEmail(c *gin.Context) {
	var input service.VerifyEmailInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	result, err := h.users.VerifyEmail(c.Request.Context(), input, sessionMeta(c))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *UserHandler) ResendEmailVerification(c *gin.Context) {
	var input service.ResendEmailVerificationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	result, err := h.users.ResendEmailVerification(c.Request.Context(), input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *UserHandler) Login(c *gin.Context) {
	var input service.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	result, err := h.users.Login(c.Request.Context(), input, sessionMeta(c))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidAuth)
		return
	}

	response.OK(c, result)
}

func (h *UserHandler) Refresh(c *gin.Context) {
	var input service.RefreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	result, err := h.users.Refresh(c.Request.Context(), input, sessionMeta(c))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *UserHandler) Logout(c *gin.Context) {
	var input service.LogoutInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	if err := h.users.Logout(c.Request.Context(), input); err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, gin.H{"logged_out": true})
}

func (h *UserHandler) LogoutAll(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}

	if err := h.users.LogoutAll(c.Request.Context(), userID.(int64)); err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, gin.H{"logged_out": true})
}

func (h *UserHandler) Me(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}

	user, err := h.users.FindByID(c.Request.Context(), userID.(int64))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, user)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}

	var input service.ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	user, err := h.users.UpdateProfile(c.Request.Context(), userID.(int64), input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, user)
}

func (h *UserHandler) List(c *gin.Context) {
	input, err := parseUserListInput(c)
	if err != nil {
		response.FromError(c, err)
		return
	}

	users, err := h.users.List(c.Request.Context(), input)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.OK(c, users)
}

func (h *UserHandler) UpdateRole(c *gin.Context) {
	actorID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}

	id, err := parseID(c.Param("id"))
	if err != nil {
		response.FromError(c, apperror.ErrInvalidID)
		return
	}

	var input service.RoleUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	if err := h.users.UpdateRole(c.Request.Context(), actorID.(int64), id, input.Role); err != nil {
		response.FromError(c, err)
		return
	}

	audit(c, h.audit, "user.role_updated", "user", id, statusMetadata(input.Role))
	response.OK(c, gin.H{"updated": true})
}

func sessionMeta(c *gin.Context) service.SessionMeta {
	return service.SessionMeta{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}
}

func parseUserListInput(c *gin.Context) (service.UserListInput, error) {
	page, err := parseOptionalInt(c.Query("page"))
	if err != nil {
		return service.UserListInput{}, apperror.New(http.StatusBadRequest, "invalid page")
	}
	pageSize, err := parseOptionalInt(c.Query("page_size"))
	if err != nil {
		return service.UserListInput{}, apperror.New(http.StatusBadRequest, "invalid page_size")
	}

	return service.UserListInput{
		Search:   c.Query("q"),
		Role:     models.UserRole(c.Query("role")),
		Page:     page,
		PageSize: pageSize,
	}, nil
}
