package database

import (
	"fmt"
	"log/slog"

	"github.com/swopcart/server/config"

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
	return db.AutoMigrate(
		&User{},
		&Session{},
	)
}
