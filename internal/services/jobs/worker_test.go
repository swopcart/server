package jobs_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/swopcart/server/internal/services/jobs"
	"github.com/swopcart/server/internal/testkit"
)

func TestWorkerPoolProcessesJobs(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Register handler
	processed := make(chan int, 5)
	handler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		id := params["id"]
		processed <- len(id) // Just use length as a simple check
		return nil
	}

	err := svc.RegisterHandler("worker_test", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Enqueue multiple jobs
	numJobs := 3
	for i := 0; i < numJobs; i++ {
		_, err := svc.EnqueueJob(
			ctx,
			"worker_test",
			jobs.WithParameters(map[string]string{"id": string(rune('a' + i))}),
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

func TestWorkerPoolBufferedQueue(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Register a handler that blocks
	startProcessing := make(chan struct{})
	processing := make(chan int, 10)

	handler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		processing <- 1
		<-startProcessing // Block until signaled
		return nil
	}

	err := svc.RegisterHandler("blocking_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Enqueue multiple jobs (more than worker count)
	// With default worker count (2), we expect 2 to start immediately and rest to queue
	numJobs := 5
	for i := 0; i < numJobs; i++ {
		_, err := svc.EnqueueJob(ctx, "blocking_job")
		if err != nil {
			t.Fatalf("Failed to enqueue job %d: %v", i, err)
		}
	}

	// Wait for workers to start processing (default is 2 workers)
	workersStarted := 0
	for workersStarted < 2 {
		select {
		case <-processing:
			workersStarted++
		case <-time.After(2 * time.Second):
			t.Fatalf("Workers did not start processing (started %d)", workersStarted)
		}
	}

	// Verify no more jobs start (remaining jobs are queued)
	select {
	case <-processing:
		t.Fatal("More jobs started than available workers")
	case <-time.After(100 * time.Millisecond):
		// Good - jobs are queued
	}

	// Unblock jobs
	close(startProcessing)

	// Wait for all jobs to complete
	timeout := time.After(5 * time.Second)
	processedCount := 2 // Already counted first 2 jobs

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

func TestMultipleQueues(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Register handlers for different queues
	defaultExecs := make(chan bool, 5)
	priorityExecs := make(chan bool, 5)

	defaultHandler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		defaultExecs <- true
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	priorityHandler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		priorityExecs <- true
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	// Register jobs
	err := svc.RegisterScheduledJob(
		ctx,
		"default_queue_job",
		"Default queue",
		"0 0 * * * *", // Never runs
		defaultHandler,
		jobs.WithQueue("default"),
	)
	if err != nil {
		t.Fatalf("Failed to register default queue job: %v", err)
	}

	err = svc.RegisterScheduledJob(
		ctx,
		"priority_queue_job",
		"Priority queue",
		"0 0 * * * *", // Never runs
		priorityHandler,
		jobs.WithQueue("priority"),
	)
	if err != nil {
		t.Fatalf("Failed to register priority queue job: %v", err)
	}

	// Enqueue jobs to both queues
	_, err = svc.EnqueueJob(ctx, "default_queue_job")
	if err != nil {
		t.Fatalf("Failed to enqueue default job: %v", err)
	}

	_, err = svc.EnqueueJob(ctx, "priority_queue_job")
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
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Register a handler that takes time to complete
	jobStarted := make(chan bool, 1)
	jobCompleted := make(chan bool, 1)

	handler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		jobStarted <- true
		time.Sleep(500 * time.Millisecond)
		jobCompleted <- true
		return nil
	}

	err := svc.RegisterHandler("slow_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Enqueue job
	_, err = svc.EnqueueJob(ctx, "slow_job")
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
