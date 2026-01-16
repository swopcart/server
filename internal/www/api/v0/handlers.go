package v0

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swopcart/server/internal/config"
)

type APIHandlers struct {
	config *config.Config
	logger *slog.Logger
	ctx    context.Context
}

func NewAPIHandlers(
	config *config.Config,
	logger *slog.Logger,
	ctx context.Context,
) (h *APIHandlers) {
	logger = logger.WithGroup("api/v0")

	h = &APIHandlers{
		config: config,
		logger: logger,
		ctx:    ctx,
	}
	return
}

func (h *APIHandlers) InstallRoutes(r *gin.RouterGroup) {
	r.GET("/ping", h.getPing)
}

func (h *APIHandlers) getPing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ping": "pong",
	})
}
