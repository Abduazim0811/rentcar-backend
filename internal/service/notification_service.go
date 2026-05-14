package service

import (
	"context"

	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
)

type NotificationService struct {
	notifications repository.NotificationRepository
}

type NotificationInput struct {
	UserID  *int64
	Title   string
	Message string
	Type    models.NotificationType
}

func NewNotificationService(notifications repository.NotificationRepository) *NotificationService {
	return &NotificationService{notifications: notifications}
}

func (s *NotificationService) Create(ctx context.Context, input NotificationInput) error {
	notification := &models.Notification{
		UserID:  input.UserID,
		Title:   input.Title,
		Message: input.Message,
		Type:    input.Type,
	}
	return s.notifications.Create(ctx, notification)
}

func (s *NotificationService) List(ctx context.Context, userID int64, unreadOnly bool) ([]models.Notification, error) {
	return s.notifications.ListByUserID(ctx, userID, unreadOnly)
}

func (s *NotificationService) CountUnread(ctx context.Context, userID int64) (int64, error) {
	return s.notifications.CountUnreadByUserID(ctx, userID)
}

func (s *NotificationService) MarkRead(ctx context.Context, id, userID int64) error {
	return s.notifications.MarkRead(ctx, id, userID)
}
