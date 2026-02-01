package cleanup

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/jobs"
)

// modelInfo describes a model to clean up
type modelInfo struct {
	name      string
	model     interface{}
	tableName string
}

func (svc *CleanupService) handleSoftDeleteCleanup(
	ctx context.Context,
	logger *slog.Logger,
	params map[string]string,
	progress jobs.ProgressReporter,
) error {
	// Parse parameters
	retentionDays, err := parseIntParam(params, "retention_days", 30)
	if err != nil {
		return fmt.Errorf("invalid retention_days: %w", err)
	}

	batchSize, err := parseIntParam(params, "batch_size", 100)
	if err != nil {
		return fmt.Errorf("invalid batch_size: %w", err)
	}

	maxDeletions, err := parseIntParam(params, "max_deletions_per_model", 10000)
	if err != nil {
		return fmt.Errorf("invalid max_deletions_per_model: %w", err)
	}

	dryRun := params["dry_run"] == "true"

	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	logger.Info("Starting soft-delete cleanup",
		"retention_days", retentionDays,
		"cutoff_date", cutoffDate,
		"batch_size", batchSize,
		"max_deletions_per_model", maxDeletions,
		"dry_run", dryRun,
	)

	// Define models to clean (in dependency order: leaf-to-root)
	models := []modelInfo{
		{"Session", &database.Session{}, "sessions"},
		{"JobExecution", &database.JobExecution{}, "job_executions"},
		{"Job", &database.Job{}, "jobs"},
		{"User", &database.User{}, "users"},
	}

	totalModels := len(models)
	totalDeleted := 0

	// Process each model
	for i, model := range models {
		progress.UpdateProgress(i, totalModels, fmt.Sprintf("Cleaning %s", model.name))

		deleted, err := svc.cleanupModel(ctx, logger, model, cutoffDate, batchSize, maxDeletions, dryRun)
		if err != nil {
			logger.Error("Failed to cleanup model",
				"model", model.name,
				"error", err,
			)
			// Continue with other models (graceful degradation)
			continue
		}

		totalDeleted += deleted
		logger.Info("Completed cleanup for model",
			"model", model.name,
			"deleted", deleted,
		)
	}

	progress.UpdateProgress(totalModels, totalModels, "Cleanup complete")

	logger.Info("Soft-delete cleanup completed",
		"total_deleted", totalDeleted,
		"models_processed", totalModels,
		"dry_run", dryRun,
	)

	return nil
}

func (svc *CleanupService) cleanupModel(
	ctx context.Context,
	logger *slog.Logger,
	model modelInfo,
	cutoffDate time.Time,
	batchSize int,
	maxDeletions int,
	dryRun bool,
) (int, error) {
	totalDeleted := 0

	for totalDeleted < maxDeletions {
		// Find batch of soft-deleted records older than cutoff
		var ids []uint
		err := svc.db.WithContext(ctx).
			Model(model.model).
			Unscoped().
			Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoffDate).
			Limit(batchSize).
			Pluck("id", &ids).
			Error

		if err != nil {
			return totalDeleted, fmt.Errorf("failed to query records: %w", err)
		}

		if len(ids) == 0 {
			break // No more records to delete
		}

		// Check if we would exceed max deletions
		batchCount := len(ids)
		if totalDeleted+batchCount > maxDeletions {
			batchCount = maxDeletions - totalDeleted
			ids = ids[:batchCount]
		}

		if dryRun {
			logger.Info("Dry run: would delete records",
				"model", model.name,
				"count", batchCount,
				"sample_ids", ids[:min(5, len(ids))],
			)
		} else {
			// Permanently delete the batch
			result := svc.db.WithContext(ctx).
				Unscoped().
				Where("id IN ?", ids).
				Delete(model.model)

			if result.Error != nil {
				return totalDeleted, fmt.Errorf("failed to delete batch: %w", result.Error)
			}

			logger.Debug("Deleted batch",
				"model", model.name,
				"count", result.RowsAffected,
			)
		}

		totalDeleted += batchCount

		if totalDeleted >= maxDeletions {
			logger.Warn("Reached max deletions limit",
				"model", model.name,
				"limit", maxDeletions,
			)
			break
		}
	}

	return totalDeleted, nil
}

func parseIntParam(params map[string]string, key string, defaultValue int) (int, error) {
	value, exists := params[key]
	if !exists {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return parsed, nil
}
