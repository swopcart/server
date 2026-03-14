package identity

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	passwordHash, err := hashPassword(DefaultAdminPassword)
	if err != nil {
		return err
	}

	user := database.User{
		UUID:     uuid.New(),
		Username: DefaultAdminUsername,
		Password: passwordHash,
		Admin:    true,
	}

	// Use ON CONFLICT DO NOTHING so a concurrent insert never aborts the transaction.
	result := svc.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&user)
	if result.Error != nil {
		svc.logger.Error("Failed to create default admin account", "err", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		svc.logger.Warn("Default admin account created - please change the password immediately",
			"username", DefaultAdminUsername)
	}

	return nil
}
