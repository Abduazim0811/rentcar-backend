package models

import "time"

type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
)

type Notification struct {
	ID        int64            `json:"id"`
	UserID    *int64           `json:"user_id,omitempty"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	Type      NotificationType `json:"type"`
	ReadAt    *time.Time       `json:"read_at"`
	CreatedAt time.Time        `json:"created_at"`
}
