package cleanup

import (
	"context"
	"log/slog"

	"github.com/swopcart/server/internal/config"
	"gorm.io/gorm"
)

type CleanupService struct {
	context context.Context
	config  *config.Config
	logger  *slog.Logger
	db      *gorm.DB
}

func NewCleanupService(
	ctx context.Context,
	config *config.Config,
	logger *slog.Logger,
	db *gorm.DB,
) (*CleanupService, error) {
	return &CleanupService{
		context: ctx,
		config:  config,
		logger:  logger,
		db:      db,
	}, nil
}
