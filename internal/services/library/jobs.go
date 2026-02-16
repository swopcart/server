package library

import (
	"context"
	"errors"
	"log/slog"

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
	}

	return errors.Join(errs...)
}

func (svc *LibraryService) libraryScanJob(
	ctx context.Context,
	logger *slog.Logger,
	params jobs.Params,
	progress jobs.ProgressReporter,
) error {
	return errors.ErrUnsupported
}
