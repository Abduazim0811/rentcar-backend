package handler

import (
	"io"
	"net/http"
	"strings"

	"car-rental-system/internal/service"
	"car-rental-system/pkg/response"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	images *service.ImageStorage
}

func NewUploadHandler(images *service.ImageStorage) *UploadHandler {
	return &UploadHandler{images: images}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	uploaded, err := h.images.Upload(c.Request.Context(), file, header)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Created(c, uploaded)
}

func (h *UploadHandler) GetImage(c *gin.Context) {
	objectName := strings.TrimSpace(c.Param("object"))
	if objectName == "" || strings.Contains(objectName, "/") || strings.Contains(objectName, "..") {
		response.Error(c, http.StatusBadRequest, "invalid image name")
		return
	}

	object, info, err := h.images.Get(c.Request.Context(), objectName)
	if err != nil {
		response.Error(c, http.StatusNotFound, "image not found")
		return
	}
	defer object.Close()

	if info.ContentType != "" {
		c.Header("Content-Type", info.ContentType)
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Status(http.StatusOK)
	if c.Request.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(c.Writer, object)
}
