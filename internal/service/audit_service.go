package service

import (
	"context"

	"car-rental-system/internal/models"
	"car-rental-system/internal/repository"
)

type AuditService struct {
	audit repository.AuditRepository
}

type AuditInput struct {
	ActorID    *int64
	Action     string
	EntityType string
	EntityID   *int64
	Metadata   string
	IPAddress  string
	UserAgent  string
}

func NewAuditService(audit repository.AuditRepository) *AuditService {
	return &AuditService{audit: audit}
}

func (s *AuditService) Create(ctx context.Context, input AuditInput) {
	if input.Metadata == "" {
		input.Metadata = "{}"
	}
	_ = s.audit.Create(ctx, &models.AuditLog{
		ActorID:    input.ActorID,
		Action:     input.Action,
		EntityType: input.EntityType,
		EntityID:   input.EntityID,
		Metadata:   input.Metadata,
		IPAddress:  input.IPAddress,
		UserAgent:  input.UserAgent,
	})
}

func (s *AuditService) List(ctx context.Context, filter repository.AuditListFilter) (*repository.AuditListResult, error) {
	return s.audit.List(ctx, filter)
}
