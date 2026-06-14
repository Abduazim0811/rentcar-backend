package repository

import (
	"context"
	"database/sql"
	"math"
	"strconv"
	"strings"
	"time"

	"car-rental-system/internal/models"
	"car-rental-system/pkg/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id int64) (*models.User, error)
	List(ctx context.Context, filter UserListFilter) (*UserListResult, error)
	SaveEmailVerification(ctx context.Context, userID int64, codeHash string, expiresAt time.Time) error
	IncrementEmailVerificationAttempts(ctx context.Context, userID int64) error
	MarkEmailVerified(ctx context.Context, userID int64) error
	UpdateProfile(ctx context.Context, user *models.User) error
	UpdateRole(ctx context.Context, id int64, role models.UserRole) error
}

type UserListFilter struct {
	Search   string
	Role     models.UserRole
	Page     int
	PageSize int
}

type UserListResult struct {
	Items      []models.User `json:"items"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}

type UserPostgresRepository struct {
	db *pgxpool.Pool
}

func NewUserPostgresRepository(db *pgxpool.Pool) *UserPostgresRepository {
	return &UserPostgresRepository{db: db}
}

func (r *UserPostgresRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (name, email, phone, password_hash, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	if user.Role == "" {
		user.Role = models.RoleCustomer
	}

	err := r.db.QueryRow(ctx, query, user.Name, user.Email, user.Phone, user.PasswordHash, user.Role).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	return mapUserUniqueError(err)
}

func (r *UserPostgresRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, name, email, phone, password_hash, role, email_verified_at,
		       email_verification_code_hash, email_verification_expires_at,
		       email_verification_sent_at, email_verification_attempts,
		       created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user, err := scanUser(r.db.QueryRow(ctx, query, email))
	if err != nil {
		return nil, mapPostgresError(err)
	}

	return user, nil
}

func (r *UserPostgresRepository) FindByID(ctx context.Context, id int64) (*models.User, error) {
	query := `
		SELECT id, name, email, phone, password_hash, role, email_verified_at,
		       email_verification_code_hash, email_verification_expires_at,
		       email_verification_sent_at, email_verification_attempts,
		       created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user, err := scanUser(r.db.QueryRow(ctx, query, id))
	if err != nil {
		return nil, mapPostgresError(err)
	}

	return user, nil
}

func (r *UserPostgresRepository) List(ctx context.Context, filter UserListFilter) (*UserListResult, error) {
	filter = normalizeUserListFilter(filter)
	where, args := buildUserListWhere(filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`+where, args...).Scan(&total); err != nil {
		return nil, mapPostgresError(err)
	}

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)

	rows, err := r.db.Query(ctx, `
		SELECT id, name, email, phone, password_hash, role, email_verified_at,
		       email_verification_code_hash, email_verification_expires_at,
		       email_verification_sent_at, email_verification_attempts,
		       created_at, updated_at
		FROM users
	`+where+`
		ORDER BY id ASC
		LIMIT $`+strconv.Itoa(limitArg)+` OFFSET $`+strconv.Itoa(offsetArg), args...)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}

	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(filter.PageSize)))
	}

	return &UserListResult{
		Items:      users,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *UserPostgresRepository) SaveEmailVerification(ctx context.Context, userID int64, codeHash string, expiresAt time.Time) error {
	result, err := r.db.Exec(ctx, `
		UPDATE users
		SET email_verification_code_hash = $1,
		    email_verification_expires_at = $2,
		    email_verification_sent_at = NOW(),
		    email_verification_attempts = 0,
		    updated_at = NOW()
		WHERE id = $3
	`, codeHash, expiresAt, userID)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}

	return nil
}

func (r *UserPostgresRepository) IncrementEmailVerificationAttempts(ctx context.Context, userID int64) error {
	result, err := r.db.Exec(ctx, `
		UPDATE users
		SET email_verification_attempts = email_verification_attempts + 1,
		    updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}

	return nil
}

func (r *UserPostgresRepository) MarkEmailVerified(ctx context.Context, userID int64) error {
	result, err := r.db.Exec(ctx, `
		UPDATE users
		SET email_verified_at = NOW(),
		    email_verification_code_hash = NULL,
		    email_verification_expires_at = NULL,
		    email_verification_sent_at = NULL,
		    email_verification_attempts = 0,
		    updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}

	return nil
}

func (r *UserPostgresRepository) UpdateProfile(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, phone = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING role, email_verified_at, created_at, updated_at
	`

	var emailVerifiedAt sql.NullTime
	err := r.db.QueryRow(ctx, query, user.Name, user.Email, user.Phone, user.ID).
		Scan(&user.Role, &emailVerifiedAt, &user.CreatedAt, &user.UpdatedAt)
	if emailVerifiedAt.Valid {
		user.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	return mapUserUniqueError(err)
}

func (r *UserPostgresRepository) UpdateRole(ctx context.Context, id int64, role models.UserRole) error {
	result, err := r.db.Exec(ctx, `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, role, id)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}

	return nil
}

func scanUser(row pgx.Row) (*models.User, error) {
	var user models.User
	var emailVerifiedAt sql.NullTime
	var verificationCodeHash sql.NullString
	var verificationExpiresAt sql.NullTime
	var verificationSentAt sql.NullTime
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.PasswordHash,
		&user.Role,
		&emailVerifiedAt,
		&verificationCodeHash,
		&verificationExpiresAt,
		&verificationSentAt,
		&user.EmailVerificationAttempts,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if emailVerifiedAt.Valid {
		user.EmailVerifiedAt = &emailVerifiedAt.Time
	}
	if verificationCodeHash.Valid {
		user.EmailVerificationCodeHash = verificationCodeHash.String
	}
	if verificationExpiresAt.Valid {
		user.EmailVerificationExpiresAt = &verificationExpiresAt.Time
	}
	if verificationSentAt.Valid {
		user.EmailVerificationSentAt = &verificationSentAt.Time
	}

	return &user, nil
}

func normalizeUserListFilter(filter UserListFilter) UserListFilter {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	filter.Search = strings.TrimSpace(filter.Search)
	return filter
}

func buildUserListWhere(filter UserListFilter) (string, []any) {
	conditions := make([]string, 0)
	args := make([]any, 0)

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		placeholder := "$" + strconv.Itoa(len(args))
		conditions = append(conditions, "(name ILIKE "+placeholder+" OR email ILIKE "+placeholder+" OR phone ILIKE "+placeholder+")")
	}
	if filter.Role != "" {
		args = append(args, filter.Role)
		conditions = append(conditions, "role = $"+strconv.Itoa(len(args)))
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func mapUserUniqueError(err error) error {
	if isPostgresConstraint(err, "23505", "users_email_key") {
		return apperror.New(409, "email is already registered")
	}
	if isPostgresConstraint(err, "23505", "idx_users_phone_unique") {
		return apperror.New(409, "phone is already registered")
	}

	return mapPostgresError(err)
}
