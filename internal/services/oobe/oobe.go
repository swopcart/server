package oobe

import (
	"context"
	"log/slog"

	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/services/identity"
	"gorm.io/gorm"
)

const ServiceName = "oobe"

type OutOfBoxService struct {
	context  context.Context
	config   *config.Config
	logger   *slog.Logger
	db       *gorm.DB
	identity *identity.IdentityService
}

func NewOutOfBoxService(
	ctx context.Context,
	config *config.Config,
	logger *slog.Logger,
	db *gorm.DB,
	identity *identity.IdentityService,
) (*OutOfBoxService, error) {
	return &OutOfBoxService{
		context:  ctx,
		config:   config,
		logger:   logger,
		db:       db,
		identity: identity,
	}, nil
}
