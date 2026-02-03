package jobs

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/swopcart/server/internal/database"
)

type workerPool struct {
	name    string
	service *JobService
	logger  *slog.Logger

	taskQueue   chan *jobTask
	workerCount int

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type jobTask struct {
	job       database.Job
	execution database.JobExecution
	handler   JobHandler
	params    Params
}

type progressReporter struct {
	service     *JobService
	executionID uint
	mu          sync.Mutex
}

func (p *progressReporter) UpdateProgress(recordsComplete, recordsTotal int, currentRecord string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx := context.Background()
	var currentRecordPtr *string
	if currentRecord != "" {
		currentRecordPtr = &currentRecord
	}

	// Update the execution record with progress
	p.service.db.WithContext(ctx).
		Model(&database.JobExecution{}).
		Where("id = ?", p.executionID).
		Updates(map[string]interface{}{
			"records_complete": recordsComplete,
			"records_total":    recordsTotal,
			"current_record":   currentRecordPtr,
		})
}

func newWorkerPool(name string, svc *JobService, workerCount int) *workerPool {
	pool := &workerPool{
		name:        name,
		service:     svc,
		logger:      svc.logger.With("queue", name),
		taskQueue:   make(chan *jobTask, 100), // Buffered queue
		workerCount: workerCount,
		stopCh:      make(chan struct{}),
	}

	// Start workers
	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	pool.logger.Info("Worker pool started", "workers", workerCount)
	return pool
}

func (p *workerPool) worker(id int) {
	defer p.wg.Done()

	logger := p.logger.With("worker_id", id)
	logger.Debug("Worker started")

	for {
		select {
		case <-p.stopCh:
			logger.Debug("Worker stopping")
			return

		case task := <-p.taskQueue:
			p.executeTask(task, logger)
		}
	}
}

func (p *workerPool) executeTask(task *jobTask, logger *slog.Logger) {
	ctx := p.service.context

	jobLogger := logger.With(
		"job", task.job.Name,
		"execution_uuid", task.execution.UUID,
	)

	jobLogger.Info("Executing job")

	// Update execution status to "running"
	startTime := time.Now()
	task.execution.Status = "running"
	task.execution.StartedAt = startTime

	if err := p.service.db.WithContext(ctx).Save(&task.execution).Error; err != nil {
		jobLogger.Error("Failed to update execution status", "err", err)
	}

	// Create progress reporter
	progress := &progressReporter{
		service:     p.service,
		executionID: task.execution.ID,
	}

	// Execute job (no timeout - jobs can run as long as needed)
	execErr := task.handler(ctx, jobLogger, task.params, progress)

	// Calculate duration
	completedAt := time.Now()
	duration := completedAt.Sub(startTime).Milliseconds()

	// Update execution record (use Updates to avoid overwriting progress fields)
	updates := map[string]interface{}{
		"completed_at": completedAt,
		"duration":     duration,
	}

	if execErr != nil {
		updates["status"] = "failed"
		errMsg := execErr.Error()
		updates["error"] = errMsg
		jobLogger.Error("Job failed", "err", execErr, "duration_ms", duration)
	} else {
		updates["status"] = "completed"
		jobLogger.Info("Job completed", "duration_ms", duration)
	}

	// Save final execution state (preserves progress fields updated during execution)
	if err := p.service.db.WithContext(ctx).
		Model(&database.JobExecution{}).
		Where("id = ?", task.execution.ID).
		Updates(updates).Error; err != nil {
		jobLogger.Error("Failed to save execution result", "err", err)
	}

	// Update job's LastRunAt
	now := time.Now()
	p.service.db.WithContext(ctx).
		Model(&database.Job{}).
		Where("id = ?", task.job.ID).
		Update("last_run_at", now)
}

func (p *workerPool) enqueue(task *jobTask) {
	// Non-blocking enqueue
	select {
	case p.taskQueue <- task:
		// Enqueued successfully
	default:
		// Queue full - log warning and block until space available
		p.logger.Warn("Task queue full, job delayed", "job", task.job.Name)
		p.taskQueue <- task
	}
}

func (p *workerPool) shutdown(timeout time.Duration) {
	p.logger.Info("Shutting down worker pool")
	close(p.stopCh)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("All workers stopped gracefully")
	case <-time.After(timeout):
		p.logger.Warn("Worker shutdown timeout exceeded, some jobs may be interrupted")
	}
}
