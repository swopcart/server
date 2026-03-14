package library

import (
	"context"
	"errors"
	"log/slog"

	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/jobs"
)

func (svc *LibraryService) RegisterJobs(jobSvc *jobs.JobService) error {
	errs := []error{
		jobSvc.RegisterScheduledJob(
			svc.context,
			"library.scan",
			"Scan library for added, removed or updated entries",
			"0 0 3 * * *",
			svc.libraryScanJob,
			jobs.WithPriority(10),
			jobs.WithQueue("default"),
			jobs.WithDefaultParameters(jobs.Params{
				"libraries": "*",
			}),
		),
		jobSvc.RegisterOnDemandJob(
			svc.context,
			"library.reimport",
			"Delete all games and reimport from library directories",
			svc.libraryReimportJob,
			jobs.WithPriority(5),
			jobs.WithQueue("default"),
			jobs.WithDefaultParameters(jobs.Params{
				"libraries": "*",
			}),
		),
	}

	return errors.Join(errs...)
}

func (svc *LibraryService) libraryScanJob(
	ctx context.Context,
	logger *slog.Logger,
	params jobs.Params,
	progress jobs.ProgressReporter,
) error {
	librariesParam := params["libraries"]

	// Get list of libraries to scan
	var libraries []database.Library
	query := svc.db.WithContext(ctx)

	if librariesParam != "*" {
		// If specific library ID provided, filter by it
		query = query.Where("id = ?", librariesParam)
	}

	if err := query.Find(&libraries).Error; err != nil {
		return err
	}

	for _, lib := range libraries {
		if err := svc.ScanLibrary(ctx, logger, &lib, progress); err != nil {
			logger.ErrorContext(ctx, "failed to scan library", "id", lib.ID, "error", err)
			// Continue scanning other libraries
		}
	}

	return nil
}

// libraryReimportJob deletes all game data and reimports everything
func (svc *LibraryService) libraryReimportJob(
	ctx context.Context,
	logger *slog.Logger,
	params jobs.Params,
	progress jobs.ProgressReporter,
) error {
	librariesParam := params["libraries"]

	// Get list of libraries to reimport
	var libraries []database.Library
	query := svc.db.WithContext(ctx)

	if librariesParam != "*" {
		// If specific library ID provided, filter by it
		query = query.Where("id = ?", librariesParam)
	}

	if err := query.Find(&libraries).Error; err != nil {
		return err
	}

	for _, lib := range libraries {
		if err := svc.ReimportLibrary(ctx, logger, &lib, progress); err != nil {
			logger.ErrorContext(ctx, "failed to reimport library", "id", lib.ID, "error", err)
			// Continue reimporting other libraries
		}
	}

	return nil
}
