package middleware

import (
	"car-rental-system/pkg/apperror"
	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := c.Get("role")
		if !ok || (role != "admin" && role != "super_admin") {
			response.FromError(c, apperror.ErrForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}

func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := c.Get("role")
		if !ok || role != "super_admin" {
			response.FromError(c, apperror.ErrForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
