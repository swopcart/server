package cleanup_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/cleanup"
	"github.com/swopcart/server/internal/services/jobs"
	"github.com/swopcart/server/internal/testkit"
	"gorm.io/gorm"
)

func TestCleanupJobRegistration(t *testing.T) {
	tk := testkit.New(t)

	// Create and register cleanup service
	cleanupSvc, err := cleanup.NewCleanupService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create cleanup service: %v", err)
	}

	err = cleanupSvc.RegisterJobs(tk.Services.Jobs)
	if err != nil {
		t.Fatalf("Failed to register cleanup jobs: %v", err)
	}

	// Verify job was registered
	var job database.Job
	err = tk.DB.Where("name = ?", "cleanup.soft_delete").First(&job).Error
	if err != nil {
		t.Fatalf("Expected cleanup job to be registered: %v", err)
	}

	// Verify job configuration
	if job.Schedule == nil || *job.Schedule != "0 0 3 * * *" {
		t.Errorf("Expected daily 3 AM schedule, got %v", job.Schedule)
	}

	if !job.Enabled {
		t.Error("Expected job to be enabled")
	}

	if job.Queue != "default" {
		t.Errorf("Expected default queue, got %s", job.Queue)
	}

	// Verify default parameters
	if job.DefaultParameters == nil {
		t.Fatal("Expected default parameters to be set")
	}

	params, err := deserializeParams(job.DefaultParameters)
	if err != nil {
		t.Fatalf("Failed to parse default parameters: %v", err)
	}

	expectedParams := map[string]string{
		"retention_days":          "30",
		"batch_size":              "100",
		"max_deletions_per_model": "10000",
		"dry_run":                 "false",
	}

	for key, expected := range expectedParams {
		if actual, ok := params[key]; !ok || actual != expected {
			t.Errorf("Expected param %s=%s, got %s", key, expected, actual)
		}
	}
}

func TestCleanupDeletesOldRecords(t *testing.T) {
	// Use WithoutTransaction so async workers can see committed data
	tk := testkit.New(t, testkit.WithoutTransaction())

	// Clean database to known state before test
	tk.ResetDB()

	// Clean database after test completes
	defer tk.ResetDB()

	// Create and register cleanup service
	cleanupSvc, err := cleanup.NewCleanupService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create cleanup service: %v", err)
	}

	err = cleanupSvc.RegisterJobs(tk.Services.Jobs)
	if err != nil {
		t.Fatalf("Failed to register cleanup jobs: %v", err)
	}

	// Create test data
	now := time.Now()
	oldDate := now.AddDate(0, 0, -31)    // 31 days ago (should be deleted)
	recentDate := now.AddDate(0, 0, -15) // 15 days ago (should NOT be deleted)

	oldUser1 := createAndSoftDeleteUser(t, tk.DB, "cleanup_old1_"+uuid.New().String()[:8], oldDate)
	oldUser2 := createAndSoftDeleteUser(t, tk.DB, "cleanup_old2_"+uuid.New().String()[:8], oldDate)
	recentUser := createAndSoftDeleteUser(t, tk.DB, "cleanup_recent_"+uuid.New().String()[:8], recentDate)

	activeUser := &database.User{
		UUID:     uuid.New(),
		Username: "cleanup_active_" + uuid.New().String()[:8],
		Password: "password",
		Admin:    false,
	}
	if err := tk.DB.Create(activeUser).Error; err != nil {
		t.Fatalf("Failed to create active user: %v", err)
	}

	// Enqueue the cleanup job
	executionUUID, err := tk.Services.Jobs.EnqueueJob(
		tk.T.Context(),
		"cleanup.soft_delete",
		jobs.WithParameters(map[string]string{
			"retention_days":          "30",
			"batch_size":              "100",
			"max_deletions_per_model": "10000",
			"dry_run":                 "false",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to enqueue cleanup job: %v", err)
	}

	// Wait for job completion
	waitForJobCompletion(t, tk.DB, executionUUID, 10*time.Second)

	// Verify old users were deleted
	var count int64
	err = tk.DB.Unscoped().Where("id IN ?", []uint{oldUser1.ID, oldUser2.ID}).
		Model(&database.User{}).Count(&count).Error
	if err != nil {
		t.Fatalf("Failed to count old users: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected old users to be deleted, but found %d", count)
	}

	// Verify recent user still exists
	var foundRecentUser database.User
	err = tk.DB.Unscoped().Where("id = ?", recentUser.ID).First(&foundRecentUser).Error
	if err != nil {
		t.Errorf("Expected recent user to still exist: %v", err)
	}

	// Verify active user still exists
	var foundActiveUser database.User
	err = tk.DB.Where("id = ?", activeUser.ID).First(&foundActiveUser).Error
	if err != nil {
		t.Errorf("Expected active user to still exist: %v", err)
	}
}

// Helper functions

func createAndSoftDeleteUser(t *testing.T, db *gorm.DB, username string, deletedAt time.Time) *database.User {
	t.Helper()

	user := &database.User{
		UUID:     uuid.New(),
		Username: username,
		Password: "password",
		Admin:    false,
	}

	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if err := db.Model(user).Update("deleted_at", deletedAt).Error; err != nil {
		t.Fatalf("Failed to soft delete user: %v", err)
	}

	return user
}

func waitForJobCompletion(t *testing.T, db *gorm.DB, executionUUID uuid.UUID, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var execution database.JobExecution
		err := db.Where("uuid = ?", executionUUID).First(&execution).Error
		if err != nil {
			t.Fatalf("Failed to find execution: %v", err)
		}

		if execution.Status == "completed" {
			return
		}

		if execution.Status == "failed" {
			errorMsg := ""
			if execution.Error != nil {
				errorMsg = *execution.Error
			}
			t.Fatalf("Job execution failed: %s", errorMsg)
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Job did not complete within %v", timeout)
}

// deserializeParams is a helper to parse job parameters for testing
func deserializeParams(data *string) (map[string]string, error) {
	if data == nil {
		return make(map[string]string), nil
	}

	var params map[string]string
	if err := json.Unmarshal([]byte(*data), &params); err != nil {
		return nil, err
	}

	return params, nil
}
