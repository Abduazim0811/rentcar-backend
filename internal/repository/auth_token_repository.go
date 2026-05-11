package repository

import (
	"context"
	"time"

	"car-rental-system/internal/models"
	"car-rental-system/pkg/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthTokenRepository interface {
	Create(ctx context.Context, token *models.AuthToken) error
	FindActiveByHash(ctx context.Context, tokenHash string) (*models.AuthToken, error)
	RevokeByHash(ctx context.Context, tokenHash string) error
	RevokeAllByUserID(ctx context.Context, userID int64) error
	Touch(ctx context.Context, id int64) error
}

type AuthTokenPostgresRepository struct {
	db *pgxpool.Pool
}

func NewAuthTokenPostgresRepository(db *pgxpool.Pool) *AuthTokenPostgresRepository {
	return &AuthTokenPostgresRepository{db: db}
}

func (r *AuthTokenPostgresRepository) Create(ctx context.Context, token *models.AuthToken) error {
	query := `
		INSERT INTO auth_tokens (user_id, token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query, token.UserID, token.TokenHash, token.UserAgent, token.IPAddress, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt)
	return mapPostgresError(err)
}

func (r *AuthTokenPostgresRepository) FindActiveByHash(ctx context.Context, tokenHash string) (*models.AuthToken, error) {
	query := `
		SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, revoked_at, last_used_at, created_at
		FROM auth_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`

	token, err := scanAuthToken(r.db.QueryRow(ctx, query, tokenHash))
	if err != nil {
		return nil, mapPostgresError(err)
	}

	return token, nil
}

func (r *AuthTokenPostgresRepository) RevokeByHash(ctx context.Context, tokenHash string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE auth_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`, tokenHash)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}

	return nil
}

func (r *AuthTokenPostgresRepository) RevokeAllByUserID(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE auth_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1
		  AND revoked_at IS NULL
	`, userID)
	return mapPostgresError(err)
}

func (r *AuthTokenPostgresRepository) Touch(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE auth_tokens SET last_used_at = $1 WHERE id = $2`, time.Now(), id)
	return mapPostgresError(err)
}

func scanAuthToken(row pgx.Row) (*models.AuthToken, error) {
	var token models.AuthToken
	if err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.UserAgent,
		&token.IPAddress,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.LastUsedAt,
		&token.CreatedAt,
	); err != nil {
		return nil, err
	}

	return &token, nil
}
