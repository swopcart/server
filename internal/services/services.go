package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/services/identity"
	"github.com/swopcart/server/internal/services/oobe"
	"gorm.io/gorm"
)

type Services struct {
	context     context.Context
	config      *config.Config
	serviceList []string

	Identity *identity.IdentityService
	OOBE     *oobe.OutOfBoxService
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
	idsmSvc, err := identity.NewIdentitySessionManager(ctx, config,
		logger.WithGroup("idsm"), db)
	if err != nil {
		return nil, newServiceInitError("oobe", err)
	}

	oobeSvc, err := oobe.NewOutOfBoxService(ctx, config,
		logger.WithGroup("oobe"), db, idsmSvc)
	if err != nil {
		return nil, newServiceInitError("oobe", err)
	}

	return &Services{
		context:  ctx,
		config:   config,
		Identity: idsmSvc,
		OOBE:     oobeSvc,
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
