package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/swopcart/server/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase(config *config.Config, logger *slog.Logger) (db *gorm.DB, err error) {
	logger = logger.WithGroup("database")

	db, err = initDatabase(config, logger)
	if err != nil {
		return
	}

	err = migrate(db)
	if err != nil {
		err = fmt.Errorf("failed to migrate database: %w", err)
		return
	}

	return
}

func initDatabase(config *config.Config, l *slog.Logger) (db *gorm.DB, err error) {
	db, err = gorm.Open(postgres.Open(config.Database.Conn), &gorm.Config{
		Logger: logger.NewSlogLogger(l, logger.Config{LogLevel: logger.Info}),
	})
	return
}

func migrate(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// Acquire a session advisory lock on a dedicated connection to serialize
	// migrations across concurrent processes/test packages, preventing DDL conflicts.
	// Using a dedicated conn ensures lock and unlock happen on the same connection.
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(8765432100)")
		_ = conn.Close()
	}()

	if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_lock(8765432100)"); err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}

	return db.AutoMigrate(Models...)
}
