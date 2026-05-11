package middleware

import (
	"context"

	"car-rental-system/internal/models"
	"car-rental-system/pkg/apperror"
	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type PermissionChecker interface {
	HasPermission(ctx context.Context, role models.UserRole, permission string) (bool, error)
}

func RequirePermission(checker PermissionChecker, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, ok := c.Get("role")
		roleString, _ := roleValue.(string)
		if !ok || roleString == "" {
			response.FromError(c, apperror.ErrForbidden)
			c.Abort()
			return
		}

		if checker == nil {
			if roleString != string(models.RoleAdmin) && roleString != string(models.RoleSuperAdmin) {
				response.FromError(c, apperror.ErrForbidden)
				c.Abort()
				return
			}
			c.Next()
			return
		}

		allowed, err := checker.HasPermission(c.Request.Context(), models.UserRole(roleString), permission)
		if err != nil || !allowed {
			response.FromError(c, apperror.ErrForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
