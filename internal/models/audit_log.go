package models

import "time"

type AuditLog struct {
	ID         int64       `json:"id"`
	ActorID    *int64      `json:"actor_id,omitempty"`
	Actor      *AuditActor `json:"actor,omitempty"`
	Action     string      `json:"action"`
	EntityType string      `json:"entity_type"`
	EntityID   *int64      `json:"entity_id,omitempty"`
	Metadata   string      `json:"metadata"`
	IPAddress  string      `json:"ip_address"`
	UserAgent  string      `json:"user_agent"`
	CreatedAt  time.Time   `json:"created_at"`
}

type AuditActor struct {
	ID    int64    `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Role  UserRole `json:"role"`
}
