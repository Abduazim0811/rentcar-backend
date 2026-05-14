package handler

import (
	"net/http"

	"car-rental-system/internal/service"
	"car-rental-system/pkg/apperror"
	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notifications *service.NotificationService
}

func NewNotificationHandler(notifications *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}
	items, err := h.notifications.List(c.Request.Context(), userID.(int64), c.Query("unread") == "true")
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *NotificationHandler) CountUnread(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}
	count, err := h.notifications.CountUnread(c.Request.Context(), userID.(int64))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"count": count})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.FromError(c, apperror.ErrUnauthorized)
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid notification id")
		return
	}
	if err := h.notifications.MarkRead(c.Request.Context(), id, userID.(int64)); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, gin.H{"read": true})
}
