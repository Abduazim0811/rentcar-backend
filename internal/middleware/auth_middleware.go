package middleware

import (
	"strings"

	"car-rental-system/internal/auth"
	"car-rental-system/pkg/apperror"
	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	jwtSecret string
}

func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{jwtSecret: jwtSecret}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.FromError(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		tokenValue := strings.TrimPrefix(header, "Bearer ")
		if tokenValue == header {
			response.FromError(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(tokenValue, m.jwtSecret)
		if err != nil {
			response.FromError(c, apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", string(claims.Role))
		c.Next()
	}
}
