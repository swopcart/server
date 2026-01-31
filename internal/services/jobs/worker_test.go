package jobs

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
)

func TestWorkerPoolCreation(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	cfg.Jobs.DefaultWorkers = 3
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Get or create a worker pool
	pool, err := svc.getOrCreateWorkerPool("test_queue")
	if err != nil {
		t.Fatalf("Failed to get worker pool: %v", err)
	}

	if pool == nil {
		t.Fatal("Expected non-nil worker pool")
	}
	if pool.name != "test_queue" {
		t.Errorf("Expected pool name 'test_queue', got %s", pool.name)
	}
	if pool.workerCount != 3 {
		t.Errorf("Expected 3 workers, got %d", pool.workerCount)
	}

	// Verify getting same pool again returns same instance
	pool2, err := svc.getOrCreateWorkerPool("test_queue")
	if err != nil {
		t.Fatalf("Failed to get worker pool second time: %v", err)
	}

	if pool != pool2 {
		t.Error("Expected same pool instance on second call")
	}
}

func TestWorkerPoolWithCustomWorkerCount(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	cfg.Jobs.DefaultWorkers = 2
	cfg.Jobs.Queues = map[string]config.QueueConfig{
		"custom_queue": {Workers: 5},
	}

	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Get custom queue
	customPool, err := svc.getOrCreateWorkerPool("custom_queue")
	if err != nil {
		t.Fatalf("Failed to get custom worker pool: %v", err)
	}

	if customPool.workerCount != 5 {
		t.Errorf("Expected 5 workers for custom queue, got %d", customPool.workerCount)
	}

	// Get default queue
	defaultPool, err := svc.getOrCreateWorkerPool("default")
	if err != nil {
		t.Fatalf("Failed to get default worker pool: %v", err)
	}

	if defaultPool.workerCount != 2 {
		t.Errorf("Expected 2 workers for default queue, got %d", defaultPool.workerCount)
	}
}

func TestWorkerPoolProcessesJobs(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	cfg.Jobs.DefaultWorkers = 1 // Single worker for predictable execution
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Register handler
	processed := make(chan int, 5)
	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		id := params["id"]
		processed <- len(id) // Just use length as a simple check
		return nil
	}

	err = svc.RegisterHandler("worker_test", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Enqueue multiple jobs
	numJobs := 3
	for i := 0; i < numJobs; i++ {
		_, err := svc.EnqueueJob(
			tk.Context(),
			"worker_test",
			WithParameters(map[string]string{"id": string(rune('a' + i))}),
		)
		if err != nil {
			t.Fatalf("Failed to enqueue job %d: %v", i, err)
		}
	}

	// Verify all jobs are processed
	timeout := time.After(5 * time.Second)
	processedCount := 0

	for processedCount < numJobs {
		select {
		case <-processed:
			processedCount++
		case <-timeout:
			t.Fatalf("Timeout waiting for jobs (processed %d of %d)", processedCount, numJobs)
		}
	}

	if processedCount != numJobs {
		t.Errorf("Expected %d jobs processed, got %d", numJobs, processedCount)
	}
}

func TestWorkerPoolShutdown(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	cfg.Jobs.DefaultWorkers = 2
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Get a worker pool
	pool, err := svc.getOrCreateWorkerPool("shutdown_test")
	if err != nil {
		t.Fatalf("Failed to get worker pool: %v", err)
	}

	// Shutdown the pool
	pool.shutdown(2 * time.Second)

	// Verify shutdown completed (this will hang if shutdown doesn't work)
	// The test passes if we get here without timeout
}

func TestWorkerPoolBufferedQueue(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	cfg.Jobs.DefaultWorkers = 1 // Single worker
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Register a handler that blocks
	startProcessing := make(chan struct{})
	processing := make(chan int, 10)

	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		processing <- 1
		<-startProcessing // Block until signaled
		return nil
	}

	err = svc.RegisterHandler("blocking_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Enqueue multiple jobs (more than worker count)
	numJobs := 5
	for i := 0; i < numJobs; i++ {
		_, err := svc.EnqueueJob(tk.Context(), "blocking_job")
		if err != nil {
			t.Fatalf("Failed to enqueue job %d: %v", i, err)
		}
	}

	// Wait for first job to start processing
	select {
	case <-processing:
		// First job started
	case <-time.After(2 * time.Second):
		t.Fatal("First job did not start processing")
	}

	// Verify other jobs are queued (not processed yet)
	select {
	case <-processing:
		t.Fatal("Second job should not start until first completes")
	case <-time.After(100 * time.Millisecond):
		// Good - jobs are queued
	}

	// Unblock jobs
	close(startProcessing)

	// Wait for all jobs to complete
	timeout := time.After(5 * time.Second)
	processedCount := 1 // Already counted first job

	for processedCount < numJobs {
		select {
		case <-processing:
			processedCount++
		case <-timeout:
			t.Fatalf("Timeout waiting for jobs (processed %d of %d)", processedCount, numJobs)
		}
	}

	if processedCount != numJobs {
		t.Errorf("Expected %d jobs processed, got %d", numJobs, processedCount)
	}
}

func TestProgressReporter(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Create a job execution record
	job := database.Job{
		UUID:        tk.UUID(),
		Name:        "progress_test",
		Description: "Test",
		Queue:       "default",
	}
	if err := tk.DB().Create(&job).Error; err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	execution := database.JobExecution{
		UUID:        tk.UUID(),
		JobID:       job.ID,
		Status:      "running",
		TriggerType: "manual",
		StartedAt:   time.Now(),
	}
	if err := tk.DB().Create(&execution).Error; err != nil {
		t.Fatalf("Failed to create execution: %v", err)
	}

	// Create progress reporter
	reporter := &progressReporter{
		service:     svc,
		executionID: execution.ID,
	}

	// Update progress
	reporter.UpdateProgress(10, 100, "Processing item 10")

	// Verify database was updated
	var updated database.JobExecution
	err = tk.DB().First(&updated, execution.ID).Error
	if err != nil {
		t.Fatalf("Failed to load updated execution: %v", err)
	}

	if updated.RecordsComplete != 10 {
		t.Errorf("Expected RecordsComplete=10, got %d", updated.RecordsComplete)
	}
	if updated.RecordsTotal != 100 {
		t.Errorf("Expected RecordsTotal=100, got %d", updated.RecordsTotal)
	}
	if updated.CurrentRecord == nil || *updated.CurrentRecord != "Processing item 10" {
		t.Errorf("Expected CurrentRecord='Processing item 10', got %v", updated.CurrentRecord)
	}

	// Update with empty current record
	reporter.UpdateProgress(20, 100, "")

	err = tk.DB().First(&updated, execution.ID).Error
	if err != nil {
		t.Fatalf("Failed to load updated execution: %v", err)
	}

	if updated.RecordsComplete != 20 {
		t.Errorf("Expected RecordsComplete=20, got %d", updated.RecordsComplete)
	}
	// CurrentRecord should be nil when empty string is passed
	if updated.CurrentRecord != nil {
		t.Errorf("Expected CurrentRecord=nil for empty string, got %v", *updated.CurrentRecord)
	}
}

func TestMultipleQueues(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	cfg.Jobs.DefaultWorkers = 1
	cfg.Jobs.Queues = map[string]config.QueueConfig{
		"priority": {Workers: 2},
	}

	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Register handlers for different queues
	defaultExecs := make(chan bool, 5)
	priorityExecs := make(chan bool, 5)

	defaultHandler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		defaultExecs <- true
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	priorityHandler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		priorityExecs <- true
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	// Register jobs
	err = svc.RegisterScheduledJob(
		tk.Context(),
		"default_queue_job",
		"Default queue",
		"0 0 * * * *", // Never runs
		defaultHandler,
		WithQueue("default"),
	)
	if err != nil {
		t.Fatalf("Failed to register default queue job: %v", err)
	}

	err = svc.RegisterScheduledJob(
		tk.Context(),
		"priority_queue_job",
		"Priority queue",
		"0 0 * * * *", // Never runs
		priorityHandler,
		WithQueue("priority"),
	)
	if err != nil {
		t.Fatalf("Failed to register priority queue job: %v", err)
	}

	// Enqueue jobs to both queues
	_, err = svc.EnqueueJob(tk.Context(), "default_queue_job")
	if err != nil {
		t.Fatalf("Failed to enqueue default job: %v", err)
	}

	_, err = svc.EnqueueJob(tk.Context(), "priority_queue_job")
	if err != nil {
		t.Fatalf("Failed to enqueue priority job: %v", err)
	}

	// Verify both queues process jobs independently
	timeout := time.After(5 * time.Second)

	gotDefault := false
	gotPriority := false

	for !gotDefault || !gotPriority {
		select {
		case <-defaultExecs:
			gotDefault = true
		case <-priorityExecs:
			gotPriority = true
		case <-timeout:
			t.Fatalf("Timeout waiting for queue executions (default=%v, priority=%v)", gotDefault, gotPriority)
		}
	}
}

func TestWorkerPoolGracefulShutdownWithRunningJobs(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	cfg.Jobs.DefaultWorkers = 1
	cfg.Jobs.ShutdownTimeout = 2 * time.Second

	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Register a handler that takes time to complete
	jobStarted := make(chan bool, 1)
	jobCompleted := make(chan bool, 1)

	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		jobStarted <- true
		time.Sleep(500 * time.Millisecond)
		jobCompleted <- true
		return nil
	}

	err = svc.RegisterHandler("slow_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Enqueue job
	_, err = svc.EnqueueJob(tk.Context(), "slow_job")
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Wait for job to start
	<-jobStarted

	// Start shutdown in background
	shutdownComplete := make(chan bool, 1)
	go func() {
		_ = svc.Shutdown()
		shutdownComplete <- true
	}()

	// Verify job completes before shutdown finishes
	select {
	case <-jobCompleted:
		// Good - job completed
	case <-time.After(3 * time.Second):
		t.Fatal("Job did not complete during graceful shutdown")
	}

	// Verify shutdown completes
	select {
	case <-shutdownComplete:
		// Good - shutdown complete
	case <-time.After(3 * time.Second):
		t.Fatal("Shutdown did not complete")
	}
}

func TestJobExecutionRecordsHostname(t *testing.T) {
	// Note: This test verifies getHostname() function works
	// even though we don't use it in the current implementation
	hostname := getHostname()
	if hostname == "" {
		t.Error("Expected non-empty hostname")
	}
	if hostname == "unknown" {
		// This is acceptable if os.Hostname() fails
		t.Skip("Hostname unavailable (acceptable)")
	}
}
