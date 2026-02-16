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
	"github.com/swopcart/server/internal/services/library"
	"github.com/swopcart/server/internal/services/session"
	"gorm.io/gorm"
)

type Services struct {
	context context.Context
	config  *config.Config

	Identity *identity.IdentityService
	Session  *session.SessionService
	Jobs     *jobs.JobService
	Library  *library.LibraryService
	Cleanup  *cleanup.CleanupService
}

type ServiceInitError struct {
	ServiceName string
}

type JobRegistrationError struct {
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

	librarySvc, err := library.NewLibraryService(ctx, config, logger.WithGroup("library"), db)
	if err != nil {
		return nil, newServiceInitError("library", err)
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
		Library:  librarySvc,
		Cleanup:  cleanupSvc,
	}, nil
}

// RegisterJobs calls RegisterJobs on each service that implements it
func (s *Services) RegisterJobs() error {
	if err := s.Library.RegisterJobs(s.Jobs); err != nil {
		return newJobRegistrationError("library", err)
	}

	if err := s.Cleanup.RegisterJobs(s.Jobs); err != nil {
		return newJobRegistrationError("cleanup", err)
	}

	return nil
}

func newServiceInitError(serviceName string, causes ...error) error {
	return errors.Join(append([]error{&ServiceInitError{ServiceName: serviceName}}, causes...)...)
}

func (err *ServiceInitError) Error() string {
	return fmt.Sprintf("couldn't initialise %s", err.ServiceName)
}

func (err *JobRegistrationError) Error() string {
	return fmt.Sprintf("failed to register %s jobs", err.ServiceName)
}

func newJobRegistrationError(serviceName string, causes ...error) error {
	return errors.Join(append([]error{&JobRegistrationError{ServiceName: serviceName}}, causes...)...)
}
