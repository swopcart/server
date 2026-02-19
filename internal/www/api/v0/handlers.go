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

	users := r.Group("/users", h.authMiddleware())
	{
		users.GET("/", h.userList)
		users.POST("/", h.userCreate)
		users.GET("/:user_uuid", h.userGetDetails)
		users.POST("/:user_uuid/password", h.userChangePassword)
		users.POST("/:user_uuid/totp/generate", h.userGenerateTOTP)
		users.POST("/:user_uuid/totp", h.userEnableTOTP)
		users.DELETE("/:user_uuid/totp", h.userDisableTOTP)
		users.GET("/:user_uuid/sessions", h.userListSessions)
	}

	jobs := r.Group("/jobs", h.authMiddleware())
	{
		jobs.GET("/", h.jobsList)
		jobs.GET("/executions/recent", h.jobsListRecentExecutions)
		jobs.GET("/executions/:execution_uuid", h.jobsGetExecution)
		jobs.GET("/:job_name", h.jobsGetDetails)
		jobs.PATCH("/:job_name", h.jobsUpdateSettings)
		jobs.GET("/:job_name/history", h.jobsGetHistory)
		jobs.POST("/:job_name/trigger", h.jobsTrigger)
	}

	platforms := r.Group("/platforms", h.authMiddleware())
	{
		platforms.GET("/", h.ListPlatformsHandler)
	}

	libraries := r.Group("/libraries", h.authMiddleware())
	{
		libraries.GET("/", h.ListLibrariesHandler)
		libraries.POST("/", h.CreateLibraryHandler)
		libraries.GET("/:libraryId", h.GetLibraryHandler)
		libraries.PATCH("/:libraryId", h.UpdateLibraryHandler)
		libraries.DELETE("/:libraryId", h.DeleteLibraryHandler)
		libraries.POST("/:libraryId/scan", h.TriggerLibraryScanHandler)
	}

	games := r.Group("/games", h.authMiddleware())
	{
		games.GET("/by-library/:libraryId", h.ListGamesHandler)
		games.GET("/:gameId", h.GetGameHandler)
		games.GET("/:gameId/versions/:versionId/download", h.DownloadGameVersionHandler)
		games.PATCH("/:gameId", h.UpdateGameMetadataHandler)
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
