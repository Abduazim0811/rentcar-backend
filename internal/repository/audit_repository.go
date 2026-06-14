package repository

import (
	"context"
	"database/sql"
	"math"
	"strconv"
	"strings"

	"car-rental-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	List(ctx context.Context, filter AuditListFilter) (*AuditListResult, error)
}

type AuditListFilter struct {
	ActorID    int64
	EntityType string
	Page       int
	PageSize   int
}

type AuditListResult struct {
	Items      []models.AuditLog `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

type AuditPostgresRepository struct {
	db *pgxpool.Pool
}

func NewAuditPostgresRepository(db *pgxpool.Pool) *AuditPostgresRepository {
	return &AuditPostgresRepository{db: db}
}

func (r *AuditPostgresRepository) Create(ctx context.Context, log *models.AuditLog) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, metadata, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7)
		RETURNING id, created_at
	`, log.ActorID, log.Action, log.EntityType, log.EntityID, log.Metadata, log.IPAddress, log.UserAgent).Scan(&log.ID, &log.CreatedAt)
	return mapPostgresError(err)
}

func (r *AuditPostgresRepository) List(ctx context.Context, filter AuditListFilter) (*AuditListResult, error) {
	filter = normalizeAuditListFilter(filter)
	where, args := buildAuditWhere(filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs al`+where, args...).Scan(&total); err != nil {
		return nil, mapPostgresError(err)
	}

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.Query(ctx, `
		SELECT al.id, al.actor_id, al.action, al.entity_type, al.entity_id, al.metadata::text, al.ip_address, al.user_agent, al.created_at,
		       u.id, u.name, u.email, u.role
		FROM audit_logs al
		LEFT JOIN users u ON u.id = al.actor_id
	`+where+`
		ORDER BY al.created_at DESC, al.id DESC
		LIMIT $`+strconv.Itoa(limitArg)+` OFFSET $`+strconv.Itoa(offsetArg), args...)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	items := make([]models.AuditLog, 0)
	for rows.Next() {
		item, err := scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(filter.PageSize)))
	}

	return &AuditListResult{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize, TotalPages: totalPages}, nil
}

func normalizeAuditListFilter(filter AuditListFilter) AuditListFilter {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	return filter
}

func buildAuditWhere(filter AuditListFilter) (string, []any) {
	args := make([]any, 0)
	conditions := make([]string, 0)
	if filter.ActorID > 0 {
		args = append(args, filter.ActorID)
		conditions = append(conditions, "al.actor_id = $"+strconv.Itoa(len(args)))
	}
	if filter.EntityType != "" {
		args = append(args, filter.EntityType)
		conditions = append(conditions, "al.entity_type = $"+strconv.Itoa(len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanAuditLog(row pgx.Row) (*models.AuditLog, error) {
	var item models.AuditLog
	var actorID sql.NullInt64
	var entityID sql.NullInt64
	var actorUserID sql.NullInt64
	var actorName sql.NullString
	var actorEmail sql.NullString
	var actorRole sql.NullString
	if err := row.Scan(
		&item.ID,
		&actorID,
		&item.Action,
		&item.EntityType,
		&entityID,
		&item.Metadata,
		&item.IPAddress,
		&item.UserAgent,
		&item.CreatedAt,
		&actorUserID,
		&actorName,
		&actorEmail,
		&actorRole,
	); err != nil {
		return nil, err
	}
	if actorID.Valid {
		item.ActorID = &actorID.Int64
	}
	if entityID.Valid {
		item.EntityID = &entityID.Int64
	}
	if actorUserID.Valid {
		item.Actor = &models.AuditActor{
			ID:    actorUserID.Int64,
			Name:  actorName.String,
			Email: actorEmail.String,
			Role:  models.UserRole(actorRole.String),
		}
	}
	return &item, nil
}
