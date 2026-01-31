package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/swopcart/server/internal/database"
)

// loadScheduledJobs loads all enabled scheduled jobs from the database
// and registers them with the cron scheduler
func (svc *JobService) loadScheduledJobs(ctx context.Context) error {
	var jobs []database.Job
	err := svc.db.WithContext(ctx).
		Where("schedule IS NOT NULL AND enabled = ?", true).
		Find(&jobs).Error
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	svc.logger.Info("Loading scheduled jobs", "count", len(jobs))

	for _, job := range jobs {
		if err := svc.addToCron(job); err != nil {
			svc.logger.Error("Failed to add job to cron", "job", job.Name, "err", err)
			// Continue loading other jobs
		}
	}

	return nil
}

// addToCron adds a job to the cron scheduler
func (svc *JobService) addToCron(job database.Job) error {
	if job.Schedule == nil || !job.Enabled {
		return nil
	}

	svc.handlersMu.RLock()
	handler, exists := svc.handlers[job.Name]
	svc.handlersMu.RUnlock()

	if !exists {
		svc.logger.Warn("Handler not found for scheduled job, skipping", "job", job.Name)
		return nil // Not an error - handler may be registered later
	}

	// Deserialize default parameters
	defaultParams, err := deserializeParams(job.DefaultParameters)
	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	// Create a closure that executes the job
	cronFunc := func() {
		ctx := context.Background()

		// Serialize parameters for this execution
		serializedParams, err := serializeParams(defaultParams)
		if err != nil {
			svc.logger.Error("Failed to serialize parameters", "job", job.Name, "err", err)
			return
		}

		executionUUID := uuid.New()
		execution := database.JobExecution{
			UUID:        executionUUID,
			JobID:       job.ID,
			Status:      "pending",
			TriggerType: "scheduled",
			Parameters:  serializedParams,
			StartedAt:   time.Now(),
		}

		if err := svc.db.WithContext(ctx).Create(&execution).Error; err != nil {
			svc.logger.Error("Failed to create execution record", "job", job.Name, "err", err)
			return
		}

		// Enqueue to worker pool
		pool, err := svc.getOrCreateWorkerPool(job.Queue)
		if err != nil {
			svc.logger.Error("Failed to get worker pool", "queue", job.Queue, "err", err)
			return
		}

		task := &jobTask{
			job:       job,
			execution: execution,
			handler:   handler,
			params:    defaultParams,
		}

		pool.enqueue(task)
	}

	_, err = svc.cron.AddFunc(*job.Schedule, cronFunc)
	if err != nil {
		return errors.Join(ErrInvalidSchedule, err)
	}

	svc.logger.Debug("Added job to cron", "job", job.Name, "schedule", *job.Schedule)
	return nil
}

// RefreshScheduledJobs reloads all scheduled jobs from the database
// Useful after jobs are modified via API
func (svc *JobService) RefreshScheduledJobs(ctx context.Context) error {
	// Stop existing cron
	cronCtx := svc.cron.Stop()
	<-cronCtx.Done()

	// Create new cron instance
	svc.cron = cron.New(cron.WithSeconds())

	// Reload jobs
	if err := svc.loadScheduledJobs(ctx); err != nil {
		return err
	}

	// Restart cron
	svc.cron.Start()

	svc.logger.Info("Refreshed scheduled jobs")
	return nil
}
