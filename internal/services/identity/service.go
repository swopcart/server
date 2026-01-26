package identity

import (
	"context"
	"log/slog"

	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
	"gorm.io/gorm"
)

type IdentityService struct {
	context     context.Context
	config      *config.Config
	logger      *slog.Logger
	db          *gorm.DB
	pendingTOTP *pendingTOTPStore
}

const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "Admin123"
)

func NewIdentitySessionManager(
	ctx context.Context,
	cfg *config.Config,
	l *slog.Logger,
	db *gorm.DB,
) (*IdentityService, error) {
	svc := &IdentityService{
		context:     ctx,
		config:      cfg,
		logger:      l,
		db:          db,
		pendingTOTP: newPendingTOTPStore(PendingTOTPTTL),
	}

	if err := svc.ensureAdminExists(ctx); err != nil {
		return nil, err
	}

	return svc, nil
}

func (svc *IdentityService) ensureAdminExists(ctx context.Context) error {
	count, err := gorm.G[database.User](svc.db).Count(ctx, "id")
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	svc.logger.Info("No users found, creating default admin account",
		"username", DefaultAdminUsername)

	_, err = svc.CreateUser(ctx, DefaultAdminUsername, DefaultAdminPassword, true)
	if err != nil {
		svc.logger.Error("Failed to create default admin account", "err", err)
		return err
	}

	svc.logger.Warn("Default admin account created - please change the password immediately",
		"username", DefaultAdminUsername)

	return nil
}
