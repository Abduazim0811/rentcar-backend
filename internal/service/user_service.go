package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"car-rental-system/internal/auth"
	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
	"car-rental-system/pkg/apperror"
)

type UserService struct {
	users           repository.UserRepository
	tokens          repository.AuthTokenRepository
	emailSender     EmailSender
	jwtSecret       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	emailVerifyTTL  time.Duration
}

const (
	emailVerificationDigits      = 6
	maxEmailVerificationAttempts = 5
	emailVerificationCooldown    = time.Minute
	defaultEmailVerificationTTL  = 10 * time.Minute
)

type RegisterInput struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type VerifyEmailInput struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type ResendEmailVerificationInput struct {
	Email string `json:"email" binding:"required,email"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RoleUpdateInput struct {
	Role models.UserRole `json:"role" binding:"required,oneof=customer admin super_admin"`
}

type UserListInput struct {
	Search   string
	Role     models.UserRole
	Page     int
	PageSize int
}

type ProfileInput struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

type AuthResponse struct {
	Token        string       `json:"token"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	User         *models.User `json:"user"`
}

type EmailVerificationResponse struct {
	Email                string `json:"email"`
	VerificationRequired bool   `json:"verification_required"`
	Message              string `json:"message"`
	ExpiresIn            int64  `json:"expires_in"`
}

type SessionMeta struct {
	UserAgent string
	IPAddress string
}

func NewUserService(
	users repository.UserRepository,
	tokens repository.AuthTokenRepository,
	emailSender EmailSender,
	jwtSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	emailVerifyTTL time.Duration,
) *UserService {
	if emailSender == nil {
		emailSender = &LoggingEmailSender{}
	}
	if emailVerifyTTL <= 0 {
		emailVerifyTTL = defaultEmailVerificationTTL
	}
	return &UserService{
		users:           users,
		tokens:          tokens,
		emailSender:     emailSender,
		jwtSecret:       jwtSecret,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		emailVerifyTTL:  emailVerifyTTL,
	}
}

func (s *UserService) Register(ctx context.Context, input RegisterInput, _ SessionMeta) (*EmailVerificationResponse, error) {
	email := normalizeEmail(input.Email)
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperror.New(400, "name is required")
	}

	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Role:         models.RoleCustomer,
	}

	if err := s.users.Create(ctx, user); err != nil {
		if apperror.StatusCode(err) == 409 {
			return nil, apperror.New(409, "email is already registered")
		}
		return nil, err
	}

	return s.sendEmailVerification(ctx, user)
}

func (s *UserService) Login(ctx context.Context, input LoginInput, meta SessionMeta) (*AuthResponse, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(input.Email))
	if err != nil {
		return nil, apperror.ErrInvalidAuth
	}

	if !auth.CheckPassword(user.PasswordHash, input.Password) {
		return nil, apperror.ErrInvalidAuth
	}
	if user.EmailVerifiedAt == nil {
		return nil, apperror.ErrEmailNotVerified
	}

	return s.createAuthResponse(ctx, user, meta)
}

func (s *UserService) Refresh(ctx context.Context, input RefreshInput, meta SessionMeta) (*AuthResponse, error) {
	tokenHash := auth.HashRefreshToken(input.RefreshToken)
	storedToken, err := s.tokens.FindActiveByHash(ctx, tokenHash)
	if err != nil {
		return nil, apperror.ErrUnauthorized
	}

	user, err := s.users.FindByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, err
	}
	if user.EmailVerifiedAt == nil {
		return nil, apperror.ErrEmailNotVerified
	}

	if err := s.tokens.RevokeByHash(ctx, tokenHash); err != nil {
		return nil, err
	}

	return s.createAuthResponse(ctx, user, meta)
}

func (s *UserService) VerifyEmail(ctx context.Context, input VerifyEmailInput, meta SessionMeta) (*AuthResponse, error) {
	email := normalizeEmail(input.Email)
	code := strings.TrimSpace(input.Code)
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, apperror.New(400, "invalid verification code")
	}
	if user.EmailVerifiedAt != nil {
		return nil, apperror.New(409, "email is already verified")
	}
	if user.EmailVerificationCodeHash == "" || user.EmailVerificationExpiresAt == nil {
		return nil, apperror.New(400, "verification code was not requested")
	}
	if time.Now().After(*user.EmailVerificationExpiresAt) {
		return nil, apperror.New(400, "verification code expired")
	}
	if user.EmailVerificationAttempts >= maxEmailVerificationAttempts {
		return nil, apperror.New(429, "too many verification attempts; request a new code")
	}

	expectedHash := hashEmailVerificationCode(email, code, s.jwtSecret)
	if subtle.ConstantTimeCompare([]byte(expectedHash), []byte(user.EmailVerificationCodeHash)) != 1 {
		_ = s.users.IncrementEmailVerificationAttempts(ctx, user.ID)
		return nil, apperror.New(400, "invalid verification code")
	}

	if err := s.users.MarkEmailVerified(ctx, user.ID); err != nil {
		return nil, err
	}

	user, err = s.users.FindByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return s.createAuthResponse(ctx, user, meta)
}

func (s *UserService) ResendEmailVerification(ctx context.Context, input ResendEmailVerificationInput) (*EmailVerificationResponse, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(input.Email))
	if err != nil {
		return nil, apperror.ErrNotFound
	}
	if user.EmailVerifiedAt != nil {
		return nil, apperror.New(409, "email is already verified")
	}
	if user.EmailVerificationSentAt != nil && time.Since(*user.EmailVerificationSentAt) < emailVerificationCooldown {
		return nil, apperror.New(429, "verification code was sent recently")
	}

	return s.sendEmailVerification(ctx, user)
}

func (s *UserService) Logout(ctx context.Context, input LogoutInput) error {
	if input.RefreshToken == "" {
		return apperror.ErrUnauthorized
	}

	if err := s.tokens.RevokeByHash(ctx, auth.HashRefreshToken(input.RefreshToken)); err != nil {
		if err == apperror.ErrNotFound {
			return nil
		}
		return err
	}

	return nil
}

func (s *UserService) LogoutAll(ctx context.Context, userID int64) error {
	return s.tokens.RevokeAllByUserID(ctx, userID)
}

func (s *UserService) FindByID(ctx context.Context, id int64) (*models.User, error) {
	return s.users.FindByID(ctx, id)
}

func (s *UserService) UpdateProfile(ctx context.Context, id int64, input ProfileInput) (*models.User, error) {
	user := &models.User{
		ID:    id,
		Name:  strings.TrimSpace(input.Name),
		Email: normalizeEmail(input.Email),
	}

	if user.Name == "" {
		return nil, apperror.New(400, "name is required")
	}

	if err := s.users.UpdateProfile(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) List(ctx context.Context, input UserListInput) (*repository.UserListResult, error) {
	if input.Role != "" &&
		input.Role != models.RoleCustomer &&
		input.Role != models.RoleAdmin &&
		input.Role != models.RoleSuperAdmin {
		return nil, apperror.New(400, "invalid user role")
	}

	return s.users.List(ctx, repository.UserListFilter{
		Search:   input.Search,
		Role:     input.Role,
		Page:     input.Page,
		PageSize: input.PageSize,
	})
}

func (s *UserService) UpdateRole(ctx context.Context, actorID, targetID int64, role models.UserRole) error {
	if actorID == targetID && role != models.RoleSuperAdmin {
		return apperror.New(400, "super admin cannot demote their own account")
	}

	return s.users.UpdateRole(ctx, targetID, role)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *UserService) createAuthResponse(ctx context.Context, user *models.User, meta SessionMeta) (*AuthResponse, error) {
	if user.EmailVerifiedAt == nil {
		return nil, apperror.ErrEmailNotVerified
	}

	accessToken, err := auth.GenerateToken(user, s.jwtSecret, s.accessTokenTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	storedToken := &models.AuthToken{
		UserID:    user.ID,
		TokenHash: auth.HashRefreshToken(refreshToken),
		UserAgent: meta.UserAgent,
		IPAddress: meta.IPAddress,
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
	}
	if err := s.tokens.Create(ctx, storedToken); err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token:        accessToken,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
		User:         user,
	}, nil
}

func (s *UserService) sendEmailVerification(ctx context.Context, user *models.User) (*EmailVerificationResponse, error) {
	code, err := generateEmailVerificationCode()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(s.emailVerifyTTL)
	codeHash := hashEmailVerificationCode(user.Email, code, s.jwtSecret)
	if err := s.users.SaveEmailVerification(ctx, user.ID, codeHash, expiresAt); err != nil {
		return nil, err
	}
	if err := s.emailSender.SendVerificationCode(ctx, user.Email, user.Name, code); err != nil {
		return nil, apperror.New(502, "could not send verification email")
	}

	return &EmailVerificationResponse{
		Email:                user.Email,
		VerificationRequired: true,
		Message:              "verification code sent",
		ExpiresIn:            int64(s.emailVerifyTTL.Seconds()),
	}, nil
}

func generateEmailVerificationCode() (string, error) {
	limit := big.NewInt(1)
	for i := 0; i < emailVerificationDigits; i++ {
		limit.Mul(limit, big.NewInt(10))
	}

	value, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%0*d", emailVerificationDigits, value.Int64()), nil
}

func hashEmailVerificationCode(email, code, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(normalizeEmail(email)))
	mac.Write([]byte(":"))
	mac.Write([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(mac.Sum(nil))
}
