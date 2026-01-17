package identity

import (
	"context"
	"log/slog"

	"github.com/swopcart/server/internal/config"
	"gorm.io/gorm"
)

const ServiceName = "idsm"

type IdentityService struct {
	context context.Context
	config  *config.Config
	logger  *slog.Logger
	db      *gorm.DB
}

func NewIdentitySessionManager(
	ctx context.Context,
	cfg *config.Config,
	l *slog.Logger,
	db *gorm.DB,
) (*IdentityService, error) {
	return &IdentityService{
		context: ctx,
		config:  cfg,
		logger:  l,
		db:      db,
	}, nil
}

func (idsm *IdentityService) Foobar() {
	_ = idsm.config
	_ = idsm.context
	_ = idsm.logger
	_ = idsm.db
}
