package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/services/identity"
	"github.com/swopcart/server/internal/services/session"
	"gorm.io/gorm"
)

type Services struct {
	context     context.Context
	config      *config.Config
	serviceList []string

	Identity *identity.IdentityService
	Session  *session.SessionService
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

	return &Services{
		context:  ctx,
		config:   config,
		Identity: identitySvc,
		Session:  sessionSvc,
	}, nil
}

func (svc *Services) IsLoaded(service string) bool {
	return slices.Contains(svc.serviceList, service)
}

func newServiceInitError(serviceName string, causes ...error) error {
	return errors.Join(append([]error{&ServiceInitError{ServiceName: serviceName}}, causes...)...)
}

func (err *ServiceInitError) Error() string {
	return fmt.Sprintf("couldn't initialise %s", err.ServiceName)
}
