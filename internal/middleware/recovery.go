package middleware

import (
	"log/slog"
	"net/http"

	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.ErrorContext(
			c.Request.Context(),
			"panic_recovered",
			slog.String("request_id", c.GetString("request_id")),
			slog.Any("panic", recovered),
		)

		response.Error(c, http.StatusInternalServerError, "internal server error")
	})
}
