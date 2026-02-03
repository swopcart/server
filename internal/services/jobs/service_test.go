package jobs_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/jobs"
	"github.com/swopcart/server/internal/testkit"
)

func TestRegisterAndExecuteHandler(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	executed := make(chan bool, 1)
	handler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		executed <- true
		return nil
	}

	// Register handler
	err := svc.RegisterHandler("test_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Enqueue and verify it executes
	_, err = svc.EnqueueJob(ctx, "test_job")
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	select {
	case <-executed:
		// Success - handler was called
	case <-time.After(2 * time.Second):
		t.Fatal("Job did not execute")
	}

	// Try to register again - should fail
	err = svc.RegisterHandler("test_job", handler)
	if err == nil {
		t.Error("Expected error when registering duplicate handler")
	}
}

func TestRegisterScheduledJob(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	handler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		return nil
	}

	err := svc.RegisterScheduledJob(
		ctx,
		"scheduled_test",
		"Test scheduled job",
		"*/5 * * * * *", // Every 5 seconds
		handler,
		jobs.WithDefaultParameters(map[string]string{"key": "value"}),
		jobs.WithPriority(10),
		jobs.WithQueue("test_queue"),
	)
	if err != nil {
		t.Fatalf("Failed to register scheduled job: %v", err)
	}

	// Verify job was created in database
	var job database.Job
	err = tk.DB.Where("name = ?", "scheduled_test").First(&job).Error
	if err != nil {
		t.Fatalf("Failed to find job in database: %v", err)
	}

	if job.Description != "Test scheduled job" {
		t.Errorf("Expected description 'Test scheduled job', got %s", job.Description)
	}
	if job.Schedule == nil || *job.Schedule != "*/5 * * * * *" {
		t.Errorf("Expected schedule '*/5 * * * * *', got %v", job.Schedule)
	}
	if !job.Enabled {
		t.Error("Expected job to be enabled")
	}
	if job.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", job.Priority)
	}
	if job.Queue != "test_queue" {
		t.Errorf("Expected queue 'test_queue', got %s", job.Queue)
	}

	// Verify default parameters are stored (implementation detail of how they're stored is hidden)
	if job.DefaultParameters == nil {
		t.Error("Expected default parameters to be stored")
	}
}

func TestRegisterScheduledJobInvalidCron(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	handler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		return nil
	}

	err := svc.RegisterScheduledJob(
		ctx,
		"invalid_cron",
		"Invalid cron",
		"not a valid cron",
		handler,
	)
	if err == nil {
		t.Fatal("Expected error for invalid cron schedule")
	}
}

func TestEnqueueJob(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Register a handler first
	executed := make(chan bool, 1)
	handler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		executed <- true
		return nil
	}

	err := svc.RegisterHandler("test_enqueue", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Enqueue the job
	executionUUID, err := svc.EnqueueJob(
		ctx,
		"test_enqueue",
		jobs.WithParameters(map[string]string{"param1": "value1"}),
	)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	if executionUUID == uuid.Nil {
		t.Fatal("Expected non-nil execution UUID")
	}

	// Verify execution was created in database
	var execution database.JobExecution
	err = tk.DB.Where("uuid = ?", executionUUID).First(&execution).Error
	if err != nil {
		t.Fatalf("Failed to find execution in database: %v", err)
	}

	if execution.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", execution.Status)
	}
	if execution.TriggerType != "manual" {
		t.Errorf("Expected trigger type 'manual', got %s", execution.TriggerType)
	}

	// Wait for execution (with timeout)
	select {
	case <-executed:
		// Job executed successfully
		// Give worker a moment to update database
		time.Sleep(100 * time.Millisecond)
	case <-time.After(5 * time.Second):
		t.Fatal("Job did not execute within 5 seconds")
	}

	// Verify execution was updated
	err = tk.DB.Where("uuid = ?", executionUUID).First(&execution).Error
	if err != nil {
		t.Fatalf("Failed to find execution after completion: %v", err)
	}

	if execution.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", execution.Status)
	}
	if execution.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
	if execution.Duration == nil {
		t.Error("Expected Duration to be set")
	}
}

func TestEnqueueJobWithDefaultParameters(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Register scheduled job with default parameters
	receivedParams := make(chan map[string]string, 1)
	handler := func(ctx context.Context, logger *slog.Logger, params jobs.Params, progress jobs.ProgressReporter) error {
		receivedParams <- params
		return nil
	}

	err := svc.RegisterScheduledJob(
		ctx,
		"test_defaults",
		"Test defaults",
		"0 0 * * * *", // Never runs automatically
		handler,
		jobs.WithDefaultParameters(map[string]string{
			"default1": "value1",
			"default2": "value2",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}

	// Enqueue with runtime parameters that override defaults
	_, err = svc.EnqueueJob(
		ctx,
		"test_defaults",
		jobs.WithParameters(map[string]string{
			"default1": "overridden",
			"runtime1": "new",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Wait for execution and verify parameter merging
	select {
	case params := <-receivedParams:
		expected := map[string]string{
			"default1": "overridden", // Runtime override
			"default2": "value2",     // From defaults
			"runtime1": "new",        // New runtime param
		}
		if diff := cmp.Diff(expected, params); diff != "" {
			t.Errorf("Params mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Job did not execute within 5 seconds")
	}
}

func TestEnqueueJobNotFound(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	_, err := svc.EnqueueJob(ctx, "nonexistent_job")
	if err == nil {
		t.Error("Expected error when enqueueing nonexistent job")
	}
}

func TestEnqueueJobHandlerNotFound(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Create a job in DB without registering a handler
	job := database.Job{
		UUID:        uuid.New(),
		Name:        "no_handler",
		Description: "No handler",
		Queue:       "default",
	}
	if err := tk.DB.Create(&job).Error; err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	_, err := svc.EnqueueJob(ctx, "no_handler")
	if err == nil {
		t.Error("Expected error when handler not registered")
	}
}

func TestGetExecution(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Create a test execution
	job := database.Job{
		UUID:        uuid.New(),
		Name:        "test_job",
		Description: "Test",
		Queue:       "default",
	}
	if err := tk.DB.Create(&job).Error; err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	executionUUID := uuid.New()
	execution := database.JobExecution{
		UUID:        executionUUID,
		JobID:       job.ID,
		Status:      "completed",
		TriggerType: "manual",
		StartedAt:   time.Now(),
	}
	if err := tk.DB.Create(&execution).Error; err != nil {
		t.Fatalf("Failed to create execution: %v", err)
	}

	// Get execution
	result, err := svc.GetExecution(ctx, executionUUID)
	if err != nil {
		t.Fatalf("Failed to get execution: %v", err)
	}

	if result.UUID != executionUUID {
		t.Errorf("Expected UUID %s, got %s", executionUUID, result.UUID)
	}
	if result.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", result.Status)
	}
}

func TestGetExecutionNotFound(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	_, err := svc.GetExecution(ctx, uuid.New())
	if err == nil {
		t.Error("Expected error when getting nonexistent execution")
	}
}

func TestGetJobExecutionHistory(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Create a job
	job := database.Job{
		UUID:        uuid.New(),
		Name:        "test_history",
		Description: "Test",
		Queue:       "default",
	}
	if err := tk.DB.Create(&job).Error; err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	// Create multiple executions
	for i := 0; i < 5; i++ {
		execution := database.JobExecution{
			UUID:        uuid.New(),
			JobID:       job.ID,
			Status:      "completed",
			TriggerType: "manual",
			StartedAt:   time.Now().Add(time.Duration(-i) * time.Hour),
		}
		if err := tk.DB.Create(&execution).Error; err != nil {
			t.Fatalf("Failed to create execution: %v", err)
		}
	}

	// Get history
	history, err := svc.GetJobExecutionHistory(ctx, "test_history", 10)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 5 {
		t.Errorf("Expected 5 executions, got %d", len(history))
	}

	// Verify ordered by started_at DESC
	for i := 1; i < len(history); i++ {
		if history[i].StartedAt.After(history[i-1].StartedAt) {
			t.Error("Expected history to be ordered by started_at DESC")
		}
	}

	// Test limit
	limited, err := svc.GetJobExecutionHistory(ctx, "test_history", 2)
	if err != nil {
		t.Fatalf("Failed to get limited history: %v", err)
	}

	if len(limited) != 2 {
		t.Errorf("Expected 2 executions, got %d", len(limited))
	}
}

func TestListRecentExecutions(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc := tk.Services.Jobs

	// Create multiple jobs
	for i := 0; i < 3; i++ {
		job := database.Job{
			UUID:        uuid.New(),
			Name:        "job_" + string(rune('a'+i)),
			Description: "Test",
			Queue:       "default",
		}
		if err := tk.DB.Create(&job).Error; err != nil {
			t.Fatalf("Failed to create job: %v", err)
		}

		// Create executions for each job
		for j := 0; j < 2; j++ {
			execution := database.JobExecution{
				UUID:        uuid.New(),
				JobID:       job.ID,
				Status:      "completed",
				TriggerType: "manual",
				StartedAt:   time.Now().Add(time.Duration(-(i*2 + j)) * time.Hour),
			}
			if err := tk.DB.Create(&execution).Error; err != nil {
				t.Fatalf("Failed to create execution: %v", err)
			}
		}
	}

	// Get recent executions
	executions, err := svc.ListRecentExecutions(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to list executions: %v", err)
	}

	if len(executions) != 6 {
		t.Errorf("Expected 6 executions, got %d", len(executions))
	}

	// Verify ordered by started_at DESC
	for i := 1; i < len(executions); i++ {
		if executions[i].StartedAt.After(executions[i-1].StartedAt) {
			t.Error("Expected executions to be ordered by started_at DESC")
		}
	}

	// Test limit
	limited, err := svc.ListRecentExecutions(ctx, 3)
	if err != nil {
		t.Fatalf("Failed to list limited executions: %v", err)
	}

	if len(limited) != 3 {
		t.Errorf("Expected 3 executions, got %d", len(limited))
	}
}

func TestShutdown(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	svc := tk.Services.Jobs

	// Shutdown should complete without error
	err := svc.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify we can call shutdown multiple times
	err = svc.Shutdown()
	if err != nil {
		t.Fatalf("Second shutdown failed: %v", err)
	}
}
