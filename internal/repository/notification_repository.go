package repository

import (
	"context"
	"database/sql"

	"car-rental-system/internal/models"
	"car-rental-system/pkg/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *models.Notification) error
	ListByUserID(ctx context.Context, userID int64, unreadOnly bool) ([]models.Notification, error)
	MarkRead(ctx context.Context, id, userID int64) error
}

type NotificationPostgresRepository struct {
	db *pgxpool.Pool
}

func NewNotificationPostgresRepository(db *pgxpool.Pool) *NotificationPostgresRepository {
	return &NotificationPostgresRepository{db: db}
}

func (r *NotificationPostgresRepository) Create(ctx context.Context, notification *models.Notification) error {
	if notification.Type == "" {
		notification.Type = models.NotificationTypeInfo
	}
	err := r.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, title, message, type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, notification.UserID, notification.Title, notification.Message, notification.Type).Scan(&notification.ID, &notification.CreatedAt)
	return mapPostgresError(err)
}

func (r *NotificationPostgresRepository) ListByUserID(ctx context.Context, userID int64, unreadOnly bool) ([]models.Notification, error) {
	query := `
		SELECT id, user_id, title, message, type, read_at, created_at
		FROM notifications
		WHERE (user_id = $1 OR user_id IS NULL)
	`
	if unreadOnly {
		query += ` AND read_at IS NULL`
	}
	query += ` ORDER BY created_at DESC LIMIT 50`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	items := make([]models.Notification, 0)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, mapPostgresError(rows.Err())
}

func (r *NotificationPostgresRepository) MarkRead(ctx context.Context, id, userID int64) error {
	result, err := r.db.Exec(ctx, `
		UPDATE notifications
		SET read_at = NOW()
		WHERE id = $1 AND (user_id = $2 OR user_id IS NULL)
	`, id, userID)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func scanNotification(row pgx.Row) (*models.Notification, error) {
	var item models.Notification
	var userID sql.NullInt64
	var readAt sql.NullTime
	if err := row.Scan(&item.ID, &userID, &item.Title, &item.Message, &item.Type, &readAt, &item.CreatedAt); err != nil {
		return nil, err
	}
	if userID.Valid {
		item.UserID = &userID.Int64
	}
	if readAt.Valid {
		item.ReadAt = &readAt.Time
	}
	return &item, nil
}
