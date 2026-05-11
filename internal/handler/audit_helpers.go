package handler

import (
	"fmt"

	"car-rental-system/internal/service"

	"github.com/gin-gonic/gin"
)

func actorIDPtr(c *gin.Context) *int64 {
	value, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	id, ok := value.(int64)
	if !ok {
		return nil
	}
	return &id
}

func actorIDValue(c *gin.Context) int64 {
	value, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	id, _ := value.(int64)
	return id
}

func audit(c *gin.Context, audits *service.AuditService, action, entityType string, entityID int64, metadata string) {
	if audits == nil {
		return
	}
	audits.Create(c.Request.Context(), service.AuditInput{
		ActorID:    actorIDPtr(c),
		Action:     action,
		EntityType: entityType,
		EntityID:   &entityID,
		Metadata:   metadata,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	})
}

func statusMetadata(status any) string {
	return fmt.Sprintf(`{"status":%q}`, fmt.Sprint(status))
}
