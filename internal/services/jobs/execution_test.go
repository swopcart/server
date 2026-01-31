package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
)

func TestJobExecutionWithDeterminateProgress(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		// Simulate processing 5 items
		for i := 1; i <= 5; i++ {
			progress.UpdateProgress(i, 5, fmt.Sprintf("Processing item %d", i))
			time.Sleep(10 * time.Millisecond)
		}
		return nil
	}

	err = svc.RegisterHandler("progress_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	executionUUID, err := svc.EnqueueJob(tk.Context(), "progress_job")
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	// Wait for execution to complete
	time.Sleep(500 * time.Millisecond)

	// Verify execution record
	var execution database.JobExecution
	err = tk.DB().Where("uuid = ?", executionUUID).First(&execution).Error
	if err != nil {
		t.Fatalf("Failed to find execution: %v", err)
	}

	if execution.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", execution.Status)
	}
	// Progress should be persisted
	if execution.RecordsComplete != 5 {
		t.Errorf("Expected RecordsComplete=5, got %d", execution.RecordsComplete)
	}
	if execution.RecordsTotal != 5 {
		t.Errorf("Expected RecordsTotal=5, got %d", execution.RecordsTotal)
	}
}

func TestJobExecutionWithIndeterminateProgress(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		steps := []string{"Initializing", "Processing data", "Finalizing"}
		for _, step := range steps {
			progress.UpdateProgress(0, 0, step)
			time.Sleep(10 * time.Millisecond)
		}
		return nil
	}

	err = svc.RegisterHandler("indeterminate_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	executionUUID, err := svc.EnqueueJob(tk.Context(), "indeterminate_job")
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	var execution database.JobExecution
	err = tk.DB().Where("uuid = ?", executionUUID).First(&execution).Error
	if err != nil {
		t.Fatalf("Failed to find execution: %v", err)
	}

	if execution.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", execution.Status)
	}
	if execution.RecordsTotal != 0 {
		t.Errorf("Expected RecordsTotal=0 for indeterminate progress, got %d", execution.RecordsTotal)
	}
}

func TestJobExecutionWithError(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	expectedErr := errors.New("simulated error")
	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		return expectedErr
	}

	err = svc.RegisterHandler("failing_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	executionUUID, err := svc.EnqueueJob(tk.Context(), "failing_job")
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	var execution database.JobExecution
	err = tk.DB().Where("uuid = ?", executionUUID).First(&execution).Error
	if err != nil {
		t.Fatalf("Failed to find execution: %v", err)
	}

	if execution.Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", execution.Status)
	}
	if execution.Error == nil || *execution.Error != expectedErr.Error() {
		t.Errorf("Expected error '%s', got %v", expectedErr.Error(), execution.Error)
	}
}

func TestJobExecutionWithParameters(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	receivedParams := make(chan map[string]string, 1)
	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		receivedParams <- params
		return nil
	}

	err = svc.RegisterHandler("param_job", handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	expectedParams := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	executionUUID, err := svc.EnqueueJob(tk.Context(), "param_job", WithParameters(expectedParams))
	if err != nil {
		t.Fatalf("Failed to enqueue job: %v", err)
	}

	select {
	case params := <-receivedParams:
		if diff := cmp.Diff(expectedParams, params); diff != "" {
			t.Errorf("Handler params mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Job did not execute")
	}

	var execution database.JobExecution
	err = tk.DB().Where("uuid = ?", executionUUID).First(&execution).Error
	if err != nil {
		t.Fatalf("Failed to find execution: %v", err)
	}

	persistedParams, _ := deserializeParams(execution.Parameters)
	if diff := cmp.Diff(expectedParams, persistedParams); diff != "" {
		t.Errorf("Persisted params mismatch (-want +got):\n%s", diff)
	}
}

func TestScheduledJobExecution(t *testing.T) {
	tk := newTestHarness(t)

	cfg := config.DefaultConfig()
	svc, err := NewJobService(tk.Context(), &cfg, tk.Logger(), tk.DB())
	if err != nil {
		t.Fatalf("Failed to create job service: %v", err)
	}

	executions := make(chan bool, 5)
	handler := func(ctx context.Context, logger *slog.Logger, params map[string]string, progress ProgressReporter) error {
		executions <- true
		return nil
	}

	err = svc.RegisterScheduledJob(
		tk.Context(),
		"scheduled_every_second",
		"Test scheduled job",
		"*/1 * * * * *", // Every second (6-field cron with seconds)
		handler,
		WithDefaultParameters(map[string]string{"from": "schedule"}),
	)
	if err != nil {
		t.Fatalf("Failed to register scheduled job: %v", err)
	}

	// Wait for at least 2 executions
	timeout := time.After(5 * time.Second)
	executionCount := 0

	for executionCount < 2 {
		select {
		case <-executions:
			executionCount++
		case <-timeout:
			t.Fatalf("Timeout waiting for scheduled executions (got %d)", executionCount)
		}
	}
}
