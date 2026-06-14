package handler

import (
	"errors"
	"net/http"
	"os"
	"time"

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

const refreshTokenCookieName = "rent_car_refresh_token"

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

	setRefreshTokenCookie(c, result)
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
		if errors.Is(err, apperror.ErrEmailNotVerified) {
			response.FromError(c, err)
			return
		}
		response.FromError(c, apperror.ErrInvalidAuth)
		return
	}

	setRefreshTokenCookie(c, result)
	response.OK(c, result)
}

func (h *UserHandler) Refresh(c *gin.Context) {
	token, err := refreshTokenFromRequest(c)
	if err != nil {
		if errors.Is(err, apperror.ErrUnauthorized) {
			response.FromError(c, err)
			return
		}
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	result, err := h.users.Refresh(c.Request.Context(), service.RefreshInput{RefreshToken: token}, sessionMeta(c))
	if err != nil {
		response.FromError(c, err)
		return
	}

	setRefreshTokenCookie(c, result)
	response.OK(c, result)
}

func (h *UserHandler) Logout(c *gin.Context) {
	token, err := refreshTokenFromRequest(c)
	if errors.Is(err, apperror.ErrUnauthorized) {
		clearRefreshTokenCookie(c)
		response.OK(c, gin.H{"logged_out": true})
		return
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, validator.Message(err))
		return
	}

	if err := h.users.Logout(c.Request.Context(), service.LogoutInput{RefreshToken: token}); err != nil {
		response.FromError(c, err)
		return
	}

	clearRefreshTokenCookie(c)
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

	clearRefreshTokenCookie(c)
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

	before, err := h.users.FindByID(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}

	if err := h.users.UpdateRole(c.Request.Context(), actorID.(int64), id, input.Role); err != nil {
		response.FromError(c, err)
		return
	}

	after := *before
	after.Role = input.Role
	audit(c, h.audit, "user.role_updated", "user", id, auditMetadata(before, &after))
	response.OK(c, gin.H{"updated": true})
}

func sessionMeta(c *gin.Context) service.SessionMeta {
	return service.SessionMeta{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}
}

func refreshTokenFromRequest(c *gin.Context) (string, error) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}

	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			return "", err
		}
	}
	if input.RefreshToken != "" {
		return input.RefreshToken, nil
	}

	token, err := c.Cookie(refreshTokenCookieName)
	if err == nil && token != "" {
		return token, nil
	}

	return "", apperror.ErrUnauthorized
}

func setRefreshTokenCookie(c *gin.Context, result *service.AuthResponse) {
	if result == nil || result.RefreshToken == "" {
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    result.RefreshToken,
		Path:     "/api/v1/auth",
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   secureCookie(c),
		SameSite: http.SameSiteLaxMode,
	})
	result.RefreshToken = ""
}

func clearRefreshTokenCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureCookie(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func secureCookie(c *gin.Context) bool {
	return os.Getenv("APP_ENV") == "production" || c.GetHeader("X-Forwarded-Proto") == "https"
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
