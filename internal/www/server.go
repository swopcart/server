package www

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/services"
	v0 "github.com/swopcart/server/internal/www/api/v0"
)

type Server struct {
	config *config.Config
	logger *slog.Logger
	ctx    context.Context

	Router     *gin.Engine
	httpServer *http.Server

	v0 *v0.APIHandlers
}

func NewServer(
	config *config.Config,
	logger *slog.Logger,
	ctx context.Context,
	services *services.Services,
) (srv *Server, err error) {
	router := gin.Default()

	httpServer := &http.Server{
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	srv = &Server{
		config: config,
		logger: logger.WithGroup("www"),

		Router:     router,
		httpServer: httpServer,

		v0: v0.NewAPIHandlers(config, logger.WithGroup("api/v0"), ctx, services),
	}

	srv.routes()
	return
}

func (srv *Server) Listen() error {
	listener, err := net.Listen(srv.config.Server.Network, srv.config.Server.Address)
	if err != nil {
		return err
	}

	srv.logger.Info("starting server", "address", listener.Addr().String())
	return srv.httpServer.Serve(listener)
}

func (srv *Server) Shutdown() error {
	srv.logger.Info("shutting down server")
	return srv.httpServer.Shutdown(srv.ctx)
}
