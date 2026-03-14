package v0

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListPlatformsHandler returns all available platforms
func (h *APIHandlers) ListPlatformsHandler(c *gin.Context) {
	platforms, err := h.services.Library.ListPlatforms(c.Request.Context())
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "failed to list platforms", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": platforms,
		"total": len(platforms),
	})
}
