package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/swopcart/server/config"
	"github.com/swopcart/server/database"
	"github.com/swopcart/server/www"
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
	_ = db // TODO

	server, err := www.NewServer(&config, slog.Default(), ctx)
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
	_ = server.Shutdown()
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
