package library

import (
	"context"
	"log/slog"

	"github.com/swopcart/server/internal/config"
	"gorm.io/gorm"
)

type LibraryService struct {
	context context.Context
	config  *config.Config
	logger  *slog.Logger
	db      *gorm.DB
}

func NewLibraryService(
	ctx context.Context,
	cfg *config.Config,
	l *slog.Logger,
	db *gorm.DB,
) (*LibraryService, error) {
	svc := &LibraryService{
		context: ctx,
		config:  cfg,
		logger:  l,
		db:      db,
	}

	// TODO: other init may need to be done

	return svc, nil
}
