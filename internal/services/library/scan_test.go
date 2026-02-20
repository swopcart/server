package library_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/library"
	"github.com/swopcart/server/internal/testkit"
	"gorm.io/gorm"
)

// mockProgressReporter implements progress reporting for tests
type mockProgressReporter struct {
	totalRecords int
	processed    int
	current      string
}

func (m *mockProgressReporter) UpdateProgress(total, processed int, current string) {
	m.totalRecords = total
	m.processed = processed
	m.current = current
}

// copyTestFixture copies a fixture directory to a temporary location
func copyTestFixture(t *testing.T, fixtureName string) string {
	// The fixture should be relative to the package directory
	// e.g. "testdata/flat_files" not "internal/services/library/testdata/flat_files"
	sourcePath := filepath.Join("testdata", fixtureName)

	// Check if file exists relative to current package
	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatalf("Failed to find fixture: %v", err)
	}

	tmpDir := t.TempDir()
	tmpLibPath := filepath.Join(tmpDir, "library")

	if err := os.MkdirAll(tmpLibPath, 0755); err != nil {
		t.Fatalf("Failed to create temp library path: %v", err)
	}

	// Copy entire fixture directory
	if err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(tmpLibPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			_ = srcFile.Close()
		}()

		dstFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer func() {
			_ = dstFile.Close()
		}()

		_, err = io.Copy(dstFile, srcFile)
		return err
	}); err != nil {
		t.Fatalf("Failed to copy fixture: %v", err)
	}

	return tmpLibPath
}

// createLibraryWithPath creates a library with the given path and uses the NES platform
func createLibraryWithPath(t *testing.T, db *gorm.DB, libPath string, platformID uint) *database.Library {
	pathsJSON, err := json.Marshal([]string{libPath})
	if err != nil {
		t.Fatalf("Failed to marshal paths: %v", err)
	}
	pathsStr := string(pathsJSON)

	lib := &database.Library{
		ID:         uuid.New(),
		Name:       "Test Library",
		Paths:      &pathsStr,
		PlatformID: platformID,
	}
	if err := db.Create(lib).Error; err != nil {
		t.Fatalf("Failed to create library: %v", err)
	}
	return lib
}

// getNESPlatform gets the NES platform (created by library service initialization)
// This helper must be called AFTER NewLibraryService initializes the platforms
func getNESPlatform(t *testing.T, db *gorm.DB) uint {
	var platform database.Platform
	if err := db.Where("name = ?", "NES").First(&platform).Error; err != nil {
		t.Fatalf("Failed to query NES platform: %v", err)
	}
	return platform.ID
}

// TestScan_NoMetadata tests scanning when metadata.toml is completely missing.
// The scanner should generate basic metadata from the filename.
func TestScan_NoMetadata(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	tmpDir := t.TempDir()
	libPath := filepath.Join(tmpDir, "library")
	if err := os.MkdirAll(libPath, 0755); err != nil {
		t.Fatalf("Failed to create library path: %v", err)
	}

	// Create a ROM file with no metadata
	romFile := filepath.Join(libPath, "TestGame.zip")
	if err := os.WriteFile(romFile, []byte("rom content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM file: %v", err)
	}

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)
	progress := &mockProgressReporter{}

	// Scan the library
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}

	// Verify game was created
	var games []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&games).Error; err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("Expected 1 game, got %d", len(games))
	}

	game := games[0]
	if game.Title != "TestGame" {
		t.Errorf("Expected title 'TestGame', got '%s'", game.Title)
	}

	// Verify version was created
	var versions []database.GameVersion
	if err := tk.DB.Where("game_id = ?", game.ID).Find(&versions).Error; err != nil {
		t.Fatalf("Failed to query versions: %v", err)
	}

	if len(versions) != 1 {
		t.Fatalf("Expected 1 version, got %d", len(versions))
	}

	if versions[0].FilePath != romFile {
		t.Errorf("Expected file path '%s', got '%s'", romFile, versions[0].FilePath)
	}
}

// TestScan_PartialMetadata tests scanning when metadata.toml exists but is incomplete.
func TestScan_PartialMetadata(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	fixture := "flat_files"
	libPath := copyTestFixture(t, fixture)

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)
	progress := &mockProgressReporter{}

	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}

	var games []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&games).Error; err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}

	if len(games) < 1 {
		t.Fatalf("Expected at least 1 game, got %d", len(games))
	}

	// Find game1
	var game1 *database.Game
	for i := range games {
		if games[i].Title == "Flat File Game" {
			game1 = &games[i]
			break
		}
	}

	if game1 == nil {
		t.Fatalf("Game 'Flat File Game' not found")
	}

	if game1.Developer == nil || *game1.Developer != "Test Developer" {
		t.Errorf("Expected developer 'Test Developer', got '%v'", game1.Developer)
	}
}

// TestScan_FullGenericMetadata tests scanning when complete metadata exists without UUID.
func TestScan_FullGenericMetadata(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	tmpDir := t.TempDir()
	libPath := filepath.Join(tmpDir, "library")
	if err := os.MkdirAll(libPath, 0755); err != nil {
		t.Fatalf("Failed to create library path: %v", err)
	}

	metadataPath := filepath.Join(libPath, "metadata.toml")
	metadataContent := `title = "Full Metadata Game"
developer = "Dev Corp"
publisher = "Pub Inc"
releaseDate = "2023-01-15"
description = "A game with complete metadata"
`
	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	rom := filepath.Join(libPath, "game.zip")
	if err := os.WriteFile(rom, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)
	progress := &mockProgressReporter{}

	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}

	var games []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&games).Error; err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("Expected 1 game, got %d", len(games))
	}

	game := games[0]
	if game.Title != "Full Metadata Game" {
		t.Errorf("Expected title 'Full Metadata Game', got '%s'", game.Title)
	}
	if game.Developer == nil || *game.Developer != "Dev Corp" {
		t.Errorf("Expected developer 'Dev Corp', got '%v'", game.Developer)
	}
	if game.Description == nil || *game.Description != "A game with complete metadata" {
		t.Errorf("Expected description, got '%v'", game.Description)
	}
}

// TestScan_InstanceMetadata tests scanning when metadata.toml has an instance UUID.
func TestScan_InstanceMetadata(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	gameID := uuid.New()
	tmpDir := t.TempDir()
	libPath := filepath.Join(tmpDir, "library")
	if err := os.MkdirAll(libPath, 0755); err != nil {
		t.Fatalf("Failed to create library path: %v", err)
	}

	// Create metadata with specific UUID
	metadataPath := filepath.Join(libPath, "metadata.toml")
	metadataContent := `uuid = "` + gameID.String() + `"
title = "UUID Game"
`
	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	rom := filepath.Join(libPath, "game.zip")
	if err := os.WriteFile(rom, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)

	// Pre-create game with the UUID
	game := &database.Game{
		ID:         gameID,
		LibraryID:  lib.ID,
		Title:      "Old Title",
		PlatformID: platformID,
	}
	if err := tk.DB.Create(game).Error; err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	progress := &mockProgressReporter{}

	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}

	// Verify game still has same UUID and was updated
	var scannedGame database.Game
	if err := tk.DB.Where("id = ?", gameID).First(&scannedGame).Error; err != nil {
		t.Fatalf("Failed to query game: %v", err)
	}

	if scannedGame.ID != gameID {
		t.Errorf("Expected game ID %s, got %s", gameID, scannedGame.ID)
	}
	if scannedGame.Title != "UUID Game" {
		t.Errorf("Expected title 'UUID Game', got '%s'", scannedGame.Title)
	}

	// Verify version was created
	var versions []database.GameVersion
	if err := tk.DB.Where("game_id = ?", gameID).Find(&versions).Error; err != nil {
		t.Fatalf("Failed to query versions: %v", err)
	}

	if len(versions) != 1 {
		t.Fatalf("Expected 1 version, got %d", len(versions))
	}
}

// TestScan_UUIDCollision tests that scanner handles UUID collision detection.
func TestScan_UUIDCollision(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	fixture := "uuid_collision"
	libPath := copyTestFixture(t, fixture)

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)

	progress := &mockProgressReporter{}

	// Scan should proceed (it logs but doesn't error on collision)
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}

	// Both games should be created (one will have the UUID, one gets a new one)
	var games []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&games).Error; err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}

	if len(games) != 2 {
		t.Fatalf("Expected 2 games, got %d", len(games))
	}
}

// TestScan_UUIDReuse tests that scanner allows UUID reuse after original game is deleted.
func TestScan_UUIDReuse(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	reuseUUID := uuid.New()
	tmpDir := t.TempDir()
	libPath := filepath.Join(tmpDir, "library")
	if err := os.MkdirAll(libPath, 0755); err != nil {
		t.Fatalf("Failed to create library path: %v", err)
	}

	// Create metadata with UUID
	metadataPath := filepath.Join(libPath, "metadata.toml")
	content := `uuid = "` + reuseUUID.String() + `"
title = "Reused Game"
`
	if err := os.WriteFile(metadataPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	// Create ROM
	if err := os.WriteFile(filepath.Join(libPath, "game.zip"), []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)

	progress := &mockProgressReporter{}

	// First scan - creates game with UUID
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("First ScanLibrary failed: %v", err)
	}

	var games []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&games).Error; err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("Expected 1 game after first scan, got %d", len(games))
	}

	// Delete the game
	if err := tk.DB.Unscoped().Delete(&games[0]).Error; err != nil {
		t.Fatalf("Failed to delete game: %v", err)
	}

	// Clear library scan state
	lib.CurrentScanJobID = nil
	if err := tk.DB.Save(lib).Error; err != nil {
		t.Fatalf("Failed to update library: %v", err)
	}

	// Second scan - should be able to reuse the UUID
	progress = &mockProgressReporter{}
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("Second ScanLibrary failed: %v", err)
	}

	var newGames []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&newGames).Error; err != nil {
		t.Fatalf("Failed to query games after second scan: %v", err)
	}

	if len(newGames) != 1 {
		t.Fatalf("Expected 1 game after second scan, got %d", len(newGames))
	}

	if newGames[0].ID != reuseUUID {
		t.Errorf("Expected UUID %s, got %s", reuseUUID, newGames[0].ID)
	}
}

// TestScan_RemovedMetadata tests that scanner regenerates metadata when metadata.toml is deleted.
func TestScan_RemovedMetadata(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	tmpDir := t.TempDir()
	libPath := filepath.Join(tmpDir, "library")
	if err := os.MkdirAll(libPath, 0755); err != nil {
		t.Fatalf("Failed to create library path: %v", err)
	}

	// Create initial metadata
	metadataPath := filepath.Join(libPath, "metadata.toml")
	metadataContent := `title = "Original Title"
developer = "Original Dev"
`
	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	rom := filepath.Join(libPath, "game.zip")
	if err := os.WriteFile(rom, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)

	progress := &mockProgressReporter{}

	// First scan - creates game
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("First ScanLibrary failed: %v", err)
	}

	var games []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&games).Error; err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("Expected 1 game, got %d", len(games))
	}

	gameID := games[0].ID

	// Delete metadata file
	if err := os.Remove(metadataPath); err != nil {
		t.Fatalf("Failed to delete metadata: %v", err)
	}

	// Clear library scan state
	lib.CurrentScanJobID = nil
	if err := tk.DB.Save(lib).Error; err != nil {
		t.Fatalf("Failed to update library: %v", err)
	}

	// Second scan - should regenerate metadata
	progress = &mockProgressReporter{}
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("Second ScanLibrary failed: %v", err)
	}

	// Verify metadata was regenerated
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Fatalf("Metadata file was not regenerated")
	}

	// Verify game still exists
	var scannedGame database.Game
	if err := tk.DB.Where("id = ?", gameID).First(&scannedGame).Error; err != nil {
		t.Fatalf("Failed to query game: %v", err)
	}
}

// TestScan_RemovedGame tests that scanner deletes game from database when entire directory is removed.
func TestScan_RemovedGame(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	fixture := "removed_game"
	libPath := copyTestFixture(t, fixture)

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)

	progress := &mockProgressReporter{}

	// First scan - creates game
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("First ScanLibrary failed: %v", err)
	}

	var games []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&games).Error; err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("Expected 1 game, got %d", len(games))
	}

	gameID := games[0].ID

	// Delete the ROM file
	romPath := filepath.Join(libPath, "game_to_remove.zip")
	if err := os.Remove(romPath); err != nil {
		t.Fatalf("Failed to delete ROM: %v", err)
	}

	// Clear library scan state
	lib.CurrentScanJobID = nil
	if err := tk.DB.Save(lib).Error; err != nil {
		t.Fatalf("Failed to update library: %v", err)
	}

	// Second scan - should soft-delete the game
	progress = &mockProgressReporter{}
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("Second ScanLibrary failed: %v", err)
	}

	// Verify game is soft-deleted
	var deletedGame database.Game
	err = tk.DB.Where("id = ?", gameID).First(&deletedGame).Error
	if err == nil {
		t.Fatalf("Expected game to be soft-deleted, but it still exists in DB")
	}

	// Verify it exists in unscoped query (soft-deleted)
	var untrackedGame database.Game
	if err := tk.DB.Unscoped().Where("id = ?", gameID).First(&untrackedGame).Error; err != nil {
		t.Fatalf("Failed to find soft-deleted game: %v", err)
	}

	if untrackedGame.DeletedAt.Time.IsZero() {
		t.Fatalf("Expected game to be soft-deleted, but DeletedAt is zero")
	}
}

// TestScan_MetadataWriteback tests that scanner writes complete instance metadata back to disk.
func TestScan_MetadataWriteback(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	tmpDir := t.TempDir()
	libPath := filepath.Join(tmpDir, "library")
	if err := os.MkdirAll(libPath, 0755); err != nil {
		t.Fatalf("Failed to create library path: %v", err)
	}

	// Create metadata without UUID
	metadataPath := filepath.Join(libPath, "metadata.toml")
	metadataContent := `title = "Game with UUID to be added"
developer = "Dev"
`
	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	rom := filepath.Join(libPath, "game.zip")
	if err := os.WriteFile(rom, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)

	progress := &mockProgressReporter{}

	// Scan - should generate UUID and write back to metadata
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}

	// Read metadata file back
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to read metadata: %v", err)
	}

	metadataStr := string(metadataBytes)

	// Verify UUID was written to metadata (it should contain "uuid = ")
	if !contains(metadataStr, "uuid") {
		t.Fatalf("Expected metadata to contain 'uuid', but got: %s", metadataStr)
	}

	// Parse metadata to verify it's valid
	metadata, err := svc.LoadMetadataFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to parse written metadata: %v", err)
	}

	if metadata == nil || metadata.Title == "" {
		t.Fatalf("Metadata is invalid after writeback")
	}
}

// contains checks if string contains substring
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
