package v0

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/services"
)

type APIHandlers struct {
	config   *config.Config
	logger   *slog.Logger
	ctx      context.Context
	services *services.Services
}

func NewAPIHandlers(
	config *config.Config,
	logger *slog.Logger,
	ctx context.Context,
	services *services.Services,
) *APIHandlers {
	return &APIHandlers{
		config:   config,
		logger:   logger,
		ctx:      ctx,
		services: services,
	}
}

func (h *APIHandlers) InstallRoutes(r *gin.RouterGroup) {
	r.GET("/ping", h.getPing)

	auth := r.Group("/auth")
	{
		auth.POST("/login", h.authLogin)
		auth.POST("/connect", h.authConnect)

		auth.POST("/refresh", h.authRefresh)
		auth.DELETE("/", h.authMiddleware(), h.authRevokeSelf)

		sessions := auth.Group("/sessions", h.authMiddleware())
		{
			sessions.GET("/", h.authListSessions)
			// sessions.DELETE("/", h.authRevokeAll)

			sessions.DELETE("/:session_uuid", h.authRevokeSession)
		}
	}
}

func (h *APIHandlers) getPing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ping": "pong",
	})
}

func (h *APIHandlers) getLogger(c *gin.Context) *slog.Logger {
	return h.logger.
		With("request_id", c.GetString("request_id")).
		With("path", c.FullPath())
}
