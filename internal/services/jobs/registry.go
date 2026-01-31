package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/swopcart/server/internal/database"
	"gorm.io/gorm"
)

// JobOption is a functional option for job configuration
type JobOption func(*database.Job)

// WithPriority sets the job priority
func WithPriority(priority int) JobOption {
	return func(j *database.Job) { j.Priority = priority }
}

// WithQueue sets the job queue
func WithQueue(queue string) JobOption {
	return func(j *database.Job) { j.Queue = queue }
}

// WithEnabled sets whether the job is enabled
func WithEnabled(enabled bool) JobOption {
	return func(j *database.Job) { j.Enabled = enabled }
}

// WithDefaultParameters sets default parameters for the job
func WithDefaultParameters(params map[string]string) JobOption {
	return func(j *database.Job) {
		serialized, _ := serializeParams(params)
		j.DefaultParameters = serialized
	}
}

// EnqueueOption is a functional option for enqueueing jobs
type EnqueueOption func(*enqueueConfig)

type enqueueConfig struct {
	queue      string
	parameters map[string]string
}

// WithQueueOverride overrides the default queue for this execution
func WithQueueOverride(queue string) EnqueueOption {
	return func(cfg *enqueueConfig) { cfg.queue = queue }
}

// WithParameters sets runtime parameters for this execution
func WithParameters(params map[string]string) EnqueueOption {
	return func(cfg *enqueueConfig) { cfg.parameters = params }
}

// RegisterHandler registers a job handler that can be invoked on-demand or scheduled
func (svc *JobService) RegisterHandler(name string, handler JobHandler) error {
	svc.handlersMu.Lock()
	defer svc.handlersMu.Unlock()

	if _, exists := svc.handlers[name]; exists {
		return ErrHandlerAlreadyRegistered
	}

	svc.handlers[name] = handler
	svc.logger.Debug("Registered job handler", "name", name)
	return nil
}

// RegisterScheduledJob registers a job with a cron schedule
func (svc *JobService) RegisterScheduledJob(
	ctx context.Context,
	name string,
	description string,
	schedule string,
	handler JobHandler,
	opts ...JobOption,
) error {
	// Register handler first
	if err := svc.RegisterHandler(name, handler); err != nil && !errors.Is(err, ErrHandlerAlreadyRegistered) {
		return err
	}

	// Parse and validate cron schedule (with seconds support)
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(schedule); err != nil {
		return errors.Join(ErrInvalidSchedule, err)
	}

	// Create or update job in database
	jobUUID := uuid.New()
	job := database.Job{
		UUID:        jobUUID,
		Name:        name,
		Description: description,
		Schedule:    &schedule,
		Enabled:     true,
		Priority:    0,
		Queue:       "default",
	}

	// Apply options
	for _, opt := range opts {
		opt(&job)
	}

	// Upsert job (update if exists, create otherwise)
	err := svc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing database.Job
		err := tx.Where("name = ?", name).First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new job
			return tx.Create(&job).Error
		} else if err != nil {
			return err
		}

		// Update existing job
		job.ID = existing.ID
		job.UUID = existing.UUID
		return tx.Save(&job).Error
	})

	if err != nil {
		return errors.Join(ErrInternal, err)
	}

	// Add to cron scheduler
	return svc.addToCron(job)
}

// EnqueueJob enqueues a job for immediate execution
func (svc *JobService) EnqueueJob(
	ctx context.Context,
	jobName string,
	opts ...EnqueueOption,
) (uuid.UUID, error) {
	// Parse options
	cfg := &enqueueConfig{
		parameters: make(map[string]string),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	svc.handlersMu.RLock()
	handler, exists := svc.handlers[jobName]
	svc.handlersMu.RUnlock()

	if !exists {
		return uuid.Nil, ErrHandlerNotFound
	}

	// Get or create job record
	var job database.Job
	err := svc.db.WithContext(ctx).
		Where("name = ?", jobName).
		First(&job).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create on-demand job entry
		job = database.Job{
			UUID:        uuid.New(),
			Name:        jobName,
			Description: "",
			Schedule:    nil, // No schedule = on-demand only
			Queue:       "default",
		}
		if err := svc.db.WithContext(ctx).Create(&job).Error; err != nil {
			return uuid.Nil, errors.Join(ErrInternal, err)
		}
	} else if err != nil {
		return uuid.Nil, errors.Join(ErrInternal, err)
	}

	// Merge runtime parameters with default parameters
	defaultParams, err := deserializeParams(job.DefaultParameters)
	if err != nil {
		return uuid.Nil, errors.Join(ErrInternal, err)
	}
	finalParams := mergeParams(defaultParams, cfg.parameters)

	// Serialize final parameters
	serializedParams, err := serializeParams(finalParams)
	if err != nil {
		return uuid.Nil, errors.Join(ErrInternal, err)
	}

	// Create execution record
	executionUUID := uuid.New()
	execution := database.JobExecution{
		UUID:        executionUUID,
		JobID:       job.ID,
		Status:      "pending",
		TriggerType: "manual",
		Parameters:  serializedParams,
		StartedAt:   time.Now(), // Will be updated when execution actually starts
	}

	if err := svc.db.WithContext(ctx).Create(&execution).Error; err != nil {
		return uuid.Nil, errors.Join(ErrInternal, err)
	}

	// Determine queue
	queueName := job.Queue
	if cfg.queue != "" {
		queueName = cfg.queue
	}

	// Enqueue to worker pool
	pool, err := svc.getOrCreateWorkerPool(queueName)
	if err != nil {
		return uuid.Nil, err
	}

	task := &jobTask{
		job:       job,
		execution: execution,
		handler:   handler,
		params:    finalParams,
	}

	pool.enqueue(task)

	svc.logger.Info("Enqueued job", "job", jobName, "execution_uuid", executionUUID)
	return executionUUID, nil
}
