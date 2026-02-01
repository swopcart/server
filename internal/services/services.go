package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/services/cleanup"
	"github.com/swopcart/server/internal/services/identity"
	"github.com/swopcart/server/internal/services/jobs"
	"github.com/swopcart/server/internal/services/session"
	"gorm.io/gorm"
)

type Services struct {
	context     context.Context
	config      *config.Config
	serviceList []string

	Identity *identity.IdentityService
	Session  *session.SessionService
	Jobs     *jobs.JobService
	Cleanup  *cleanup.CleanupService
}

type ServiceInitError struct {
	ServiceName string
}

func NewServices(
	ctx context.Context,
	config *config.Config,
	logger *slog.Logger,
	db *gorm.DB,
) (*Services, error) {
	identitySvc, err := identity.NewIdentitySessionManager(ctx, config,
		logger.WithGroup("identity"), db)
	if err != nil {
		return nil, newServiceInitError("identity", err)
	}

	sessionSvc, err := session.NewSessionService(config, logger.WithGroup("session"), db, identitySvc)
	if err != nil {
		return nil, newServiceInitError("session", err)
	}

	jobsSvc, err := jobs.NewJobService(ctx, config, logger.WithGroup("jobs"), db)
	if err != nil {
		return nil, newServiceInitError("jobs", err)
	}

	cleanupSvc, err := cleanup.NewCleanupService(ctx, config, logger.WithGroup("cleanup"), db)
	if err != nil {
		return nil, newServiceInitError("cleanup", err)
	}

	return &Services{
		context:  ctx,
		config:   config,
		Identity: identitySvc,
		Session:  sessionSvc,
		Jobs:     jobsSvc,
		Cleanup:  cleanupSvc,
	}, nil
}

func newServiceInitError(serviceName string, causes ...error) error {
	return errors.Join(append([]error{&ServiceInitError{ServiceName: serviceName}}, causes...)...)
}

func (err *ServiceInitError) Error() string {
	return fmt.Sprintf("couldn't initialise %s", err.ServiceName)
}
