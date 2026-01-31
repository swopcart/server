package jobs

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
)

func TestNewJobService(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	cfg.Jobs.DefaultWorkers = 2

	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	if svc == nil {
		t.Fatal("Expected non-nil service")
	}

	// Verify cron is started
	if svc.cron == nil {
		t.Fatal("Expected cron scheduler to be initialized")
	}

	// Verify maps are initialized
	if svc.handlers == nil {
		t.Fatal("Expected handlers map to be initialized")
	}
	if svc.workers == nil {
		t.Fatal("Expected workers map to be initialized")
	}

	// Verify default worker pool exists
	svc.workersMu.RLock()
	defaultPool := svc.workers["default"]
	svc.workersMu.RUnlock()

	if defaultPool == nil {
		t.Fatal("Expected default worker pool to be created")
	}
	if defaultPool.workerCount != 2 {
		t.Errorf("Expected 2 workers, got %d", defaultPool.workerCount)
	}
}

func TestRegisterHandler(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		return nil
	}

	err = svc.RegisterHandler("test_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Verify handler is registered
	svc.handlersMu.RLock()
	registered := svc.handlers["test_job"]
	svc.handlersMu.RUnlock()

	if registered == nil {
		t.Fatal("Expected handler to be registered")
	}

	// Try to register again - should fail
	err = svc.RegisterHandler("test_job", handler)
	if err != ErrHandlerAlreadyRegistered {
		t.Errorf("Expected ErrHandlerAlreadyRegistered, got %v", err)
	}
}

func TestRegisterScheduledJob(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		return nil
	}

	err = svc.RegisterScheduledJob(
		tk.Context(),
		"scheduled_test",
		"Test scheduled job",
		"*/5 * * * * *", // Every 5 seconds
		handler,
		WithDefaultParameters(map[string]string{"key": "value"}),
		WithPriority(10),
		WithQueue("test_queue"),
	)
	if err != nil {
		t.Fatalf("Failed to register scheduled job: %v", err)
	}

	// Verify job was created in database
	var job database.Job
	err = tk.DB().Where("name = ?", "scheduled_test").First(&job).Error
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

	// Verify default parameters
	params, err := deserializeParams(job.DefaultParameters)
	if err != nil {
		t.Fatalf("Failed to deserialize params: %v", err)
	}
	if params["key"] != "value" {
		t.Errorf("Expected param key=value, got %v", params)
	}

	// Verify handler was registered
	svc.handlersMu.RLock()
	registered := svc.handlers["scheduled_test"]
	svc.handlersMu.RUnlock()

	if registered == nil {
		t.Fatal("Expected handler to be registered")
	}
}

func TestRegisterScheduledJobInvalidCron(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		return nil
	}

	err = svc.RegisterScheduledJob(
		tk.Context(),
		"invalid_cron",
		"Invalid cron",
		"not a valid cron",
		handler,
	)
	if err == nil {
		t.Fatal("Expected error for invalid cron schedule")
	}
}

func TestSerializeDeserializeParams(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]string
	}{
		{
			name:   "empty",
			params: map[string]string{},
		},
		{
			name:   "single param",
			params: map[string]string{"key": "value"},
		},
		{
			name: "multiple params",
			params: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
		{
			name: "special characters",
			params: map[string]string{
				"path":  "/path/to/file",
				"query": "key=value&foo=bar",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized, err := serializeParams(tt.params)
			if err != nil {
				t.Fatalf("Failed to serialize: %v", err)
			}

			deserialized, err := deserializeParams(serialized)
			if err != nil {
				t.Fatalf("Failed to deserialize: %v", err)
			}

			if diff := cmp.Diff(tt.params, deserialized); diff != "" {
				t.Errorf("Params mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeParams(t *testing.T) {
	tests := []struct {
		name     string
		defaults map[string]string
		runtime  map[string]string
		expected map[string]string
	}{
		{
			name:     "no defaults, no runtime",
			defaults: map[string]string{},
			runtime:  map[string]string{},
			expected: map[string]string{},
		},
		{
			name:     "only defaults",
			defaults: map[string]string{"key1": "default1", "key2": "default2"},
			runtime:  map[string]string{},
			expected: map[string]string{"key1": "default1", "key2": "default2"},
		},
		{
			name:     "only runtime",
			defaults: map[string]string{},
			runtime:  map[string]string{"key1": "runtime1"},
			expected: map[string]string{"key1": "runtime1"},
		},
		{
			name:     "runtime overrides defaults",
			defaults: map[string]string{"key1": "default1", "key2": "default2"},
			runtime:  map[string]string{"key1": "runtime1"},
			expected: map[string]string{"key1": "runtime1", "key2": "default2"},
		},
		{
			name:     "runtime adds new keys",
			defaults: map[string]string{"key1": "default1"},
			runtime:  map[string]string{"key2": "runtime2"},
			expected: map[string]string{"key1": "default1", "key2": "runtime2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeParams(tt.defaults, tt.runtime)
			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("Merged params mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEnqueueJob(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Register a handler first
	executed := make(chan bool, 1)
	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		executed <- true
		return nil
	}

	err = svc.RegisterHandler("test_enqueue", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// Enqueue the job
	executionUUID, err := svc.EnqueueJob(
		tk.Context(),
		"test_enqueue",
		WithParameters(map[string]string{"param1": "value1"}),
	)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	if executionUUID == uuid.Nil {
		t.Fatal("Expected non-nil execution UUID")
	}

	// Verify execution was created in database
	var execution database.JobExecution
	err = tk.DB().Where("uuid = ?", executionUUID).First(&execution).Error
	if err != nil {
		t.Fatalf("Failed to find execution in database: %v", err)
	}

	if execution.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", execution.Status)
	}
	if execution.TriggerType != "manual" {
		t.Errorf("Expected trigger type 'manual', got %s", execution.TriggerType)
	}

	// Verify parameters
	params, err := deserializeParams(execution.Parameters)
	if err != nil {
		t.Fatalf("Failed to deserialize params: %v", err)
	}
	if params["param1"] != "value1" {
		t.Errorf("Expected param1=value1, got %v", params)
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
	err = tk.DB().Where("uuid = ?", executionUUID).First(&execution).Error
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
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Register scheduled job with default parameters
	receivedParams := make(chan map[string]string, 1)
	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		receivedParams <- params
		return nil
	}

	err = svc.RegisterScheduledJob(
		tk.Context(),
		"test_defaults",
		"Test defaults",
		"0 0 * * * *", // Never runs automatically
		handler,
		WithDefaultParameters(map[string]string{
			"default1": "value1",
			"default2": "value2",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to register job: %v", err)
	}

	// Enqueue with runtime parameters that override defaults
	_, err = svc.EnqueueJob(
		tk.Context(),
		"test_defaults",
		WithParameters(map[string]string{
			"default1": "overridden",
			"runtime1": "new",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Wait for execution
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
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	_, err = svc.EnqueueJob(tk.Context(), "nonexistent_job")
	if err != ErrHandlerNotFound {
		t.Errorf("Expected ErrHandlerNotFound, got %v", err)
	}
}

func TestEnqueueJobHandlerNotFound(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Create a job in DB without registering a handler
	job := database.Job{
		UUID:        uuid.New(),
		Name:        "no_handler",
		Description: "No handler",
		Queue:       "default",
	}
	if err := tk.DB().Create(&job).Error; err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	_, err = svc.EnqueueJob(tk.Context(), "no_handler")
	if err != ErrHandlerNotFound {
		t.Errorf("Expected ErrHandlerNotFound, got %v", err)
	}
}

func TestGetExecution(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Create a test execution
	job := database.Job{
		UUID:        uuid.New(),
		Name:        "test_job",
		Description: "Test",
		Queue:       "default",
	}
	if err := tk.DB().Create(&job).Error; err != nil {
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
	if err := tk.DB().Create(&execution).Error; err != nil {
		t.Fatalf("Failed to create execution: %v", err)
	}

	// Get execution
	result, err := svc.GetExecution(tk.Context(), executionUUID)
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
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	_, err = svc.GetExecution(tk.Context(), uuid.New())
	if err != ErrExecutionNotFound {
		t.Errorf("Expected ErrExecutionNotFound, got %v", err)
	}
}

func TestGetJobExecutionHistory(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Create a job
	job := database.Job{
		UUID:        uuid.New(),
		Name:        "test_history",
		Description: "Test",
		Queue:       "default",
	}
	if err := tk.DB().Create(&job).Error; err != nil {
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
		if err := tk.DB().Create(&execution).Error; err != nil {
			t.Fatalf("Failed to create execution: %v", err)
		}
	}

	// Get history
	history, err := svc.GetJobExecutionHistory(tk.Context(), "test_history", 10)
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
	limited, err := svc.GetJobExecutionHistory(tk.Context(), "test_history", 2)
	if err != nil {
		t.Fatalf("Failed to get limited history: %v", err)
	}

	if len(limited) != 2 {
		t.Errorf("Expected 2 executions, got %d", len(limited))
	}
}

func TestListRecentExecutions(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Create multiple jobs
	for i := 0; i < 3; i++ {
		job := database.Job{
			UUID:        uuid.New(),
			Name:        "job_" + string(rune('a'+i)),
			Description: "Test",
			Queue:       "default",
		}
		if err := tk.DB().Create(&job).Error; err != nil {
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
			if err := tk.DB().Create(&execution).Error; err != nil {
				t.Fatalf("Failed to create execution: %v", err)
			}
		}
	}

	// Get recent executions
	executions, err := svc.ListRecentExecutions(tk.Context(), 10)
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
	limited, err := svc.ListRecentExecutions(tk.Context(), 3)
	if err != nil {
		t.Fatalf("Failed to list limited executions: %v", err)
	}

	if len(limited) != 3 {
		t.Errorf("Expected 3 executions, got %d", len(limited))
	}
}

func TestShutdown(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	cfg.Jobs.ShutdownTimeout = 5 * time.Second

	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	// Verify cron is running
	if svc.cron == nil {
		t.Fatal("Expected cron to be initialized")
	}

	// Shutdown
	err = svc.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify we can call shutdown multiple times
	err = svc.Shutdown()
	if err != nil {
		t.Fatalf("Second shutdown failed: %v", err)
	}
}
