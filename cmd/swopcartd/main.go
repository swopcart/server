package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services"
	"github.com/swopcart/server/internal/www"
)

var ErrMainExit = errors.New("main() exited")
var ErrConnClosed = errors.New("(*www.Server).Listen() exited")

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	config, err := loadConfig()
	if err != nil {
		slog.Error("can't load config", "err", err)
		os.Exit(1)
	}
	slog.Info("loaded config", "config", config)

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(ErrMainExit)

	db, err := database.NewDatabase(&config, slog.Default())
	if err != nil {
		slog.Error("can't connect to database", "err", err)
		os.Exit(1)
	}

	svc, err := services.NewServices(ctx, &config, slog.Default(), db)
	if err != nil {
		slog.Error("Failed to initialise services", "err", err)
		os.Exit(1)
	}

	// Register jobs from all services
	if err := registerJobs(svc); err != nil {
		slog.Error("Failed to register jobs", "err", err)
		os.Exit(1)
	}

	server, err := www.NewServer(&config, slog.Default(), ctx, svc)
	if err != nil {
		slog.Error("can't create web server", "err", err)
		os.Exit(1)
	}

	go func() {
		err := server.Listen()
		cancel(errors.Join(ErrConnClosed, err))
	}()

	waitForExitSignal()
	slog.Info("exiting")

	// Shutdown jobs first (prevent new jobs from being queued)
	if err := svc.Jobs.Shutdown(); err != nil {
		slog.Error("Error shutting down job service", "err", err)
	}

	_ = server.Shutdown()
}

// registerJobs calls RegisterJobs on each service that implements it
func registerJobs(svc *services.Services) error {
	// Currently no services have jobs to register
	// Services can add RegisterJobs methods as needed

	// Example for when services implement RegisterJobs:
	// if err := svc.Session.RegisterJobs(svc.Jobs); err != nil {
	//     return fmt.Errorf("session jobs: %w", err)
	// }

	return nil
}

func loadConfig() (config.Config, error) {
	slog.Info("loading config")
	configPath := filepath.Join(config.DataDir(), "config.toml")
	return config.LoadConfig(configPath)
}

func waitForExitSignal() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan
}
