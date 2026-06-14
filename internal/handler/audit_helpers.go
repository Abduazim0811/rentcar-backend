package handler

import (
	"encoding/json"
	"reflect"
	"sort"

	"car-rental-system/internal/service"

	"github.com/gin-gonic/gin"
)

type auditChange struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

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

func auditMetadata(before any, after any) string {
	payload := map[string]any{}
	if before != nil {
		payload["before"] = before
	}
	if after != nil {
		payload["after"] = after
	}
	if changes := auditChanges(before, after); len(changes) > 0 {
		payload["changes"] = changes
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func auditChanges(before any, after any) []auditChange {
	beforeMap, beforeOK := auditMap(before)
	afterMap, afterOK := auditMap(after)
	if !beforeOK || !afterOK {
		return nil
	}

	ignored := map[string]bool{
		"created_at": true,
		"updated_at": true,
	}
	keys := make(map[string]bool, len(beforeMap)+len(afterMap))
	for key := range beforeMap {
		if !ignored[key] {
			keys[key] = true
		}
	}
	for key := range afterMap {
		if !ignored[key] {
			keys[key] = true
		}
	}

	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)

	changes := make([]auditChange, 0)
	for _, key := range ordered {
		beforeValue := beforeMap[key]
		afterValue := afterMap[key]
		if reflect.DeepEqual(beforeValue, afterValue) {
			continue
		}
		changes = append(changes, auditChange{Field: key, Before: beforeValue, After: afterValue})
	}
	return changes
}

func auditMap(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, false
	}
	return result, true
}
