package response

import (
	"net/http"

	"car-rental-system/pkg/apperror"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error":      message,
		"request_id": c.GetString("request_id"),
	})
}

func FromError(c *gin.Context, err error) {
	Error(c, apperror.StatusCode(err), apperror.Message(err))
}
