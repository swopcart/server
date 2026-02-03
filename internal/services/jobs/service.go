package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
	"gorm.io/gorm"
)

// Params is a convenience alias for job parameters, similar to gin.H
type Params map[string]string

// JobHandler is the function signature for job execution
type JobHandler func(ctx context.Context, logger *slog.Logger, params Params, progress ProgressReporter) error

// ProgressReporter allows jobs to report their progress
type ProgressReporter interface {
	UpdateProgress(recordsComplete, recordsTotal int, currentRecord string)
}

// JobService manages background job execution
type JobService struct {
	context context.Context
	config  *config.Config
	logger  *slog.Logger
	db      *gorm.DB

	// Scheduling
	cron *cron.Cron

	// Job registration
	handlers   map[string]JobHandler
	handlersMu sync.RWMutex

	// Worker pools (one per queue)
	workers   map[string]*workerPool
	workersMu sync.RWMutex

	// Shutdown coordination
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
}

// NewJobService creates a new job service
func NewJobService(
	ctx context.Context,
	cfg *config.Config,
	logger *slog.Logger,
	db *gorm.DB,
) (*JobService, error) {
	svc := &JobService{
		context:    ctx,
		config:     cfg,
		logger:     logger,
		db:         db,
		handlers:   make(map[string]JobHandler),
		workers:    make(map[string]*workerPool),
		shutdownCh: make(chan struct{}),
	}

	// Initialize cron scheduler with second-level precision
	svc.cron = cron.New(cron.WithSeconds())

	// Load scheduled jobs from database and register with cron
	if err := svc.loadScheduledJobs(ctx); err != nil {
		return nil, err
	}

	// Start default worker pool
	if _, err := svc.ensureWorkerPool("default"); err != nil {
		return nil, err
	}

	// Start cron scheduler
	svc.cron.Start()
	svc.logger.Info("Job service started")

	return svc, nil
}

// DB returns the database connection for direct queries
func (svc *JobService) DB() *gorm.DB {
	return svc.db
}

// Shutdown gracefully stops the job service
func (svc *JobService) Shutdown() error {
	svc.shutdownOnce.Do(func() {
		svc.logger.Info("Shutting down job service")

		// Stop cron scheduler (no more scheduled jobs)
		cronCtx := svc.cron.Stop()
		<-cronCtx.Done()
		svc.logger.Debug("Cron scheduler stopped")

		// Signal shutdown
		close(svc.shutdownCh)

		// Shutdown all worker pools with timeout
		timeout := svc.config.Jobs.ShutdownTimeout

		svc.workersMu.Lock()
		var wg sync.WaitGroup
		for _, pool := range svc.workers {
			wg.Add(1)
			go func(p *workerPool) {
				defer wg.Done()
				p.shutdown(timeout)
			}(pool)
		}
		svc.workersMu.Unlock()

		// Wait for all pools to shutdown
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			svc.logger.Info("Job service shut down gracefully")
		case <-time.After(timeout):
			svc.logger.Warn("Shutdown timeout exceeded")
		}
	})

	return nil
}

// GetJobExecutionHistory returns recent executions for a job
func (svc *JobService) GetJobExecutionHistory(
	ctx context.Context,
	jobName string,
	limit int,
) ([]database.JobExecution, error) {
	var job database.Job
	if err := svc.db.WithContext(ctx).Where("name = ?", jobName).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	var executions []database.JobExecution
	err := svc.db.WithContext(ctx).
		Where("job_id = ?", job.ID).
		Order("started_at DESC").
		Limit(limit).
		Find(&executions).Error

	return executions, err
}

// GetExecution retrieves a specific job execution by UUID
func (svc *JobService) GetExecution(
	ctx context.Context,
	executionUUID uuid.UUID,
) (*database.JobExecution, error) {
	var execution database.JobExecution
	err := svc.db.WithContext(ctx).
		Where("uuid = ?", executionUUID).
		First(&execution).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrExecutionNotFound
	}

	return &execution, err
}

// ListRecentExecutions returns recent executions across all jobs
func (svc *JobService) ListRecentExecutions(
	ctx context.Context,
	limit int,
) ([]database.JobExecution, error) {
	var executions []database.JobExecution
	err := svc.db.WithContext(ctx).
		Order("started_at DESC").
		Limit(limit).
		Find(&executions).Error

	return executions, err
}

// Helper functions for parameter serialization

func serializeParams(params Params) (*string, error) {
	if len(params) == 0 {
		return nil, nil
	}

	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	str := string(data)
	return &str, nil
}

func deserializeParams(data *string) (Params, error) {
	if data == nil {
		return make(map[string]string), nil
	}

	var params map[string]string
	if err := json.Unmarshal([]byte(*data), &params); err != nil {
		return nil, err
	}

	return params, nil
}

// mergeParams merges runtime parameters with default parameters
// Runtime parameters override defaults
func mergeParams(defaults, runtime Params) Params {
	merged := make(map[string]string)

	// Copy defaults
	for k, v := range defaults {
		merged[k] = v
	}

	// Override with runtime
	for k, v := range runtime {
		merged[k] = v
	}

	return merged
}
