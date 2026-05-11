package models

import "time"

type AuthToken struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	TokenHash  string     `json:"-"`
	UserAgent  string     `json:"user_agent"`
	IPAddress  string     `json:"ip_address"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}
