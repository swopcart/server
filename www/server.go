package www

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/swopcart/server/config"
)

type Server struct {
	config *config.Config
	logger *slog.Logger
	ctx    context.Context

	router     *gin.Engine
	httpServer *http.Server
}

func NewServer(
	config *config.Config, logger *slog.Logger, ctx context.Context) (srv *Server, err error) {
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

		router:     router,
		httpServer: httpServer,
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
