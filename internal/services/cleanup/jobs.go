package cleanup

import (
	"github.com/swopcart/server/internal/services/jobs"
)

func (svc *CleanupService) RegisterJobs(jobSvc *jobs.JobService) error {
	// Register soft-delete cleanup job
	err := jobSvc.RegisterScheduledJob(
		svc.context,
		"cleanup.soft_delete",
		"Permanently delete soft-deleted records older than retention period",
		"0 0 3 * * *", // Daily at 3 AM
		svc.handleSoftDeleteCleanup,
		jobs.WithPriority(-10), // Low priority
		jobs.WithQueue("default"),
		jobs.WithDefaultParameters(jobs.Params{
			"retention_days":          "30",
			"batch_size":              "100",
			"max_deletions_per_model": "10000",
			"dry_run":                 "false",
		}),
	)

	return err
}
