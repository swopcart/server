package library_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/library"
	"github.com/swopcart/server/internal/testkit"
	"gorm.io/gorm"
)

func TestCreateLibrary_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Get a platform first
	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected at least one platform to exist: %v", err)
	}

	tmpDir := t.TempDir()

	req := library.CreateLibraryRequest{
		Name:        "Test Library",
		Description: "A test library",
		PlatformID:  platform.ID,
		Paths:       []string{tmpDir},
	}

	lib, err := librarySvc.CreateLibrary(tk.T.Context(), req)
	if err != nil {
		t.Fatalf("Failed to create library: %v", err)
	}

	if lib.Name != req.Name {
		t.Errorf("Expected name %q, got %q", req.Name, lib.Name)
	}

	if lib.Description != req.Description {
		t.Errorf("Expected description %q, got %q", req.Description, lib.Description)
	}

	if lib.PlatformID != platform.ID {
		t.Errorf("Expected platform ID %d, got %d", platform.ID, lib.PlatformID)
	}

	// Verify paths were stored
	var paths []string
	if err := json.Unmarshal([]byte(*lib.Paths), &paths); err != nil {
		t.Fatalf("Failed to unmarshal paths: %v", err)
	}

	if len(paths) != 1 || paths[0] != tmpDir {
		t.Errorf("Expected paths %v, got %v", []string{tmpDir}, paths)
	}

	// Verify in database
	var dbLib database.Library
	if err := tk.DB.First(&dbLib, lib.ID).Error; err != nil {
		t.Fatalf("Expected library to be in database: %v", err)
	}

	if dbLib.ScanStatus != "idle" {
		t.Errorf("Expected scan status to be idle, got %s", dbLib.ScanStatus)
	}
}

func TestCreateLibrary_InvalidPlatform(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	tmpDir := t.TempDir()

	req := library.CreateLibraryRequest{
		Name:       "Test Library",
		PlatformID: 99999, // Non-existent platform
		Paths:      []string{tmpDir},
	}

	_, err = librarySvc.CreateLibrary(tk.T.Context(), req)
	if err == nil {
		t.Fatal("Expected error for non-existent platform")
	}
}

func TestCreateLibrary_PathNotFound(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected at least one platform to exist: %v", err)
	}

	req := library.CreateLibraryRequest{
		Name:       "Test Library",
		PlatformID: platform.ID,
		Paths:      []string{"/nonexistent/path/that/does/not/exist"},
	}

	_, err = librarySvc.CreateLibrary(tk.T.Context(), req)
	if err == nil {
		t.Fatal("Expected error for non-existent path")
	}
}

func TestListLibraries_Empty(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	libraries, err := librarySvc.ListLibraries(tk.T.Context())
	if err != nil {
		t.Fatalf("Failed to list libraries: %v", err)
	}

	if len(libraries) != 0 {
		t.Errorf("Expected 0 libraries, got %d", len(libraries))
	}
}

func TestListLibraries_WithCounts(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Create a library
	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected at least one platform to exist: %v", err)
	}

	tmpDir := t.TempDir()
	req := library.CreateLibraryRequest{
		Name:       "Test Library",
		PlatformID: platform.ID,
		Paths:      []string{tmpDir},
	}

	lib, err := librarySvc.CreateLibrary(tk.T.Context(), req)
	if err != nil {
		t.Fatalf("Failed to create library: %v", err)
	}

	// Create a game manually
	game := &database.Game{
		ID:         uuid.New(),
		LibraryID:  lib.ID,
		PlatformID: platform.ID,
		Title:      "Test Game",
	}
	if err := tk.DB.Create(game).Error; err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Create a version
	version := &database.GameVersion{
		ID:          uuid.New(),
		GameID:      game.ID,
		VersionName: "1.0",
		FilePath:    "/tmp/test.rom",
		FileSize:    1024,
	}
	if err := tk.DB.Create(version).Error; err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	// List libraries
	libraries, err := librarySvc.ListLibraries(tk.T.Context())
	if err != nil {
		t.Fatalf("Failed to list libraries: %v", err)
	}

	if len(libraries) != 1 {
		t.Fatalf("Expected 1 library, got %d", len(libraries))
	}

	if libraries[0].GameCount != 1 {
		t.Errorf("Expected game count 1, got %d", libraries[0].GameCount)
	}
}

func TestGetLibrary_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected at least one platform to exist: %v", err)
	}

	tmpDir := t.TempDir()
	req := library.CreateLibraryRequest{
		Name:       "Test Library",
		PlatformID: platform.ID,
		Paths:      []string{tmpDir},
	}

	created, err := librarySvc.CreateLibrary(tk.T.Context(), req)
	if err != nil {
		t.Fatalf("Failed to create library: %v", err)
	}

	retrieved, err := librarySvc.GetLibrary(tk.T.Context(), created.ID)
	if err != nil {
		t.Fatalf("Failed to get library: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Expected ID %v, got %v", created.ID, retrieved.ID)
	}

	if retrieved.Name != created.Name {
		t.Errorf("Expected name %q, got %q", created.Name, retrieved.Name)
	}
}

func TestGetLibrary_NotFound(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	nonExistentID := uuid.New()
	_, err = librarySvc.GetLibrary(tk.T.Context(), nonExistentID)
	if err == nil {
		t.Fatal("Expected error for non-existent library")
	}
}

func TestUpdateLibrary_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected at least one platform to exist: %v", err)
	}

	tmpDir := t.TempDir()
	req := library.CreateLibraryRequest{
		Name:       "Original Name",
		PlatformID: platform.ID,
		Paths:      []string{tmpDir},
	}

	created, err := librarySvc.CreateLibrary(tk.T.Context(), req)
	if err != nil {
		t.Fatalf("Failed to create library: %v", err)
	}

	newName := "Updated Name"
	newDesc := "Updated Description"
	updateReq := library.UpdateLibraryRequest{
		Name:        &newName,
		Description: &newDesc,
	}

	updated, err := librarySvc.UpdateLibrary(tk.T.Context(), created.ID, updateReq)
	if err != nil {
		t.Fatalf("Failed to update library: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("Expected name %q, got %q", newName, updated.Name)
	}

	if updated.Description != newDesc {
		t.Errorf("Expected description %q, got %q", newDesc, updated.Description)
	}
}

func TestUpdateLibrary_NewPaths(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected at least one platform to exist: %v", err)
	}

	tmpDir1 := t.TempDir()
	req := library.CreateLibraryRequest{
		Name:       "Test Library",
		PlatformID: platform.ID,
		Paths:      []string{tmpDir1},
	}

	created, err := librarySvc.CreateLibrary(tk.T.Context(), req)
	if err != nil {
		t.Fatalf("Failed to create library: %v", err)
	}

	tmpDir2 := t.TempDir()
	updateReq := library.UpdateLibraryRequest{
		Paths: []string{tmpDir1, tmpDir2},
	}

	updated, err := librarySvc.UpdateLibrary(tk.T.Context(), created.ID, updateReq)
	if err != nil {
		t.Fatalf("Failed to update library: %v", err)
	}

	var paths []string
	if err := json.Unmarshal([]byte(*updated.Paths), &paths); err != nil {
		t.Fatalf("Failed to unmarshal paths: %v", err)
	}

	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}
}

func TestDeleteLibrary_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected at least one platform to exist: %v", err)
	}

	tmpDir := t.TempDir()
	req := library.CreateLibraryRequest{
		Name:       "Test Library",
		PlatformID: platform.ID,
		Paths:      []string{tmpDir},
	}

	created, err := librarySvc.CreateLibrary(tk.T.Context(), req)
	if err != nil {
		t.Fatalf("Failed to create library: %v", err)
	}

	// Delete library
	err = librarySvc.DeleteLibrary(tk.T.Context(), created.ID)
	if err != nil {
		t.Fatalf("Failed to delete library: %v", err)
	}

	// Verify soft delete
	var dbLib database.Library
	result := tk.DB.First(&dbLib, created.ID)
	if result.Error != gorm.ErrRecordNotFound {
		t.Fatal("Expected library to be soft-deleted (not found in non-deleted query)")
	}

	// Verify it exists in soft-deleted records
	result = tk.DB.Unscoped().First(&dbLib, created.ID)
	if result.Error != nil {
		t.Fatalf("Expected library to exist in soft-deleted records: %v", result.Error)
	}

	if dbLib.DeletedAt.Time.IsZero() {
		t.Error("Expected DeletedAt to be set")
	}
}
