package models

import "time"

type UserRole string

const (
	RoleCustomer   UserRole = "customer"
	RoleAdmin      UserRole = "admin"
	RoleSuperAdmin UserRole = "super_admin"
)

type User struct {
	ID                         int64      `json:"id"`
	Name                       string     `json:"name"`
	Email                      string     `json:"email"`
	Phone                      string     `json:"phone"`
	PasswordHash               string     `json:"-"`
	Role                       UserRole   `json:"role"`
	EmailVerifiedAt            *time.Time `json:"email_verified_at,omitempty"`
	EmailVerificationCodeHash  string     `json:"-"`
	EmailVerificationExpiresAt *time.Time `json:"-"`
	EmailVerificationSentAt    *time.Time `json:"-"`
	EmailVerificationAttempts  int        `json:"-"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
}
