package repository

import (
	"context"

	"car-rental-system/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PermissionRepository interface {
	HasPermission(ctx context.Context, role models.UserRole, permission string) (bool, error)
}

type PermissionPostgresRepository struct {
	db *pgxpool.Pool
}

func NewPermissionPostgresRepository(db *pgxpool.Pool) *PermissionPostgresRepository {
	return &PermissionPostgresRepository{db: db}
}

func (r *PermissionPostgresRepository) HasPermission(ctx context.Context, role models.UserRole, permission string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM role_permissions rp
			JOIN permissions p ON p.id = rp.permission_id
			WHERE rp.role = $1 AND p.code = $2
		)
	`, role, permission).Scan(&ok)
	return ok, mapPostgresError(err)
}
