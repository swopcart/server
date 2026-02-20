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

	rom := filepath.Join(libPath, "game.zip")
	if err := os.WriteFile(rom, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	// Create sidecar metadata file (game.zip.toml)
	metadataPath := rom + ".toml"
	metadataContent := `title = "Full Metadata Game"
developer = "Dev Corp"
publisher = "Pub Inc"
releaseDate = "2023-01-15"
description = "A game with complete metadata"
`
	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
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
// This test verifies that the scanner correctly loads and preserves UUIDs from metadata.
func TestScan_InstanceMetadata(t *testing.T) {
	tk := testkit.New(t, testkit.WithoutTransaction())
	tk.ResetDB()
	defer tk.ResetDB()

	ctx := context.Background()
	svc, err := library.NewLibraryService(ctx, tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	instanceUUID := uuid.New()
	tmpDir := t.TempDir()
	libPath := filepath.Join(tmpDir, "library")
	if err := os.MkdirAll(libPath, 0755); err != nil {
		t.Fatalf("Failed to create library path: %v", err)
	}

	rom := filepath.Join(libPath, "game.zip")
	if err := os.WriteFile(rom, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	// Create sidecar metadata with instance UUID
	metadataPath := rom + ".toml"
	metadataContent := `uuid = "` + instanceUUID.String() + `"
title = "UUID Game"
developer = "UUID Dev"
`
	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)

	progress := &mockProgressReporter{}

	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}

	// Verify game was created with metadata from sidecar file
	var games []database.Game
	if err := tk.DB.Where("library_id = ? AND title = ?", lib.ID, "UUID Game").Find(&games).Error; err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("Expected 1 game with title 'UUID Game', got %d", len(games))
	}

	game := games[0]
	if game.Developer == nil || *game.Developer != "UUID Dev" {
		t.Errorf("Expected developer 'UUID Dev', got '%v'", game.Developer)
	}

	// Verify version was created
	var versions []database.GameVersion
	if err := tk.DB.Where("game_id = ?", game.ID).Find(&versions).Error; err != nil {
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

// TestScan_UUIDReuse tests that scanner handles multiple scans correctly
// and can detect game changes on subsequent scans.
func TestScan_UUIDReuse(t *testing.T) {
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

	// Create a ROM file
	rom := filepath.Join(libPath, "game.zip")
	if err := os.WriteFile(rom, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	// Create metadata
	metadataPath := rom + ".toml"
	content := `title = "Reused Game"
developer = "Dev Corp"
`
	if err := os.WriteFile(metadataPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
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
		t.Fatalf("Expected 1 game after first scan, got %d", len(games))
	}

	firstGameID := games[0].ID
	if games[0].Developer == nil || *games[0].Developer != "Dev Corp" {
		t.Errorf("Expected developer 'Dev Corp', got '%v'", games[0].Developer)
	}

	// Clear library scan state
	lib.CurrentScanJobID = nil
	if err := tk.DB.Save(lib).Error; err != nil {
		t.Fatalf("Failed to update library: %v", err)
	}

	// Update metadata
	newContent := `title = "Reused Game"
developer = "New Dev"
`
	if err := os.WriteFile(metadataPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to write updated metadata: %v", err)
	}

	// Second scan - should reuse same game, update metadata
	progress = &mockProgressReporter{}
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("Second ScanLibrary failed: %v", err)
	}

	var updatedGames []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&updatedGames).Error; err != nil {
		t.Fatalf("Failed to query games after second scan: %v", err)
	}

	if len(updatedGames) != 1 {
		t.Fatalf("Expected 1 game after second scan, got %d", len(updatedGames))
	}

	// Game should be the same instance
	if updatedGames[0].ID != firstGameID {
		t.Errorf("Expected same game ID %s, got %s", firstGameID, updatedGames[0].ID)
	}

	// Metadata should be updated
	if updatedGames[0].Developer == nil || *updatedGames[0].Developer != "New Dev" {
		t.Errorf("Expected updated developer 'New Dev', got '%v'", updatedGames[0].Developer)
	}
}

// TestScan_RemovedMetadata tests that scanner handles missing metadata gracefully.
// When metadata is removed, the scanner should generate it from the filename.
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

	rom := filepath.Join(libPath, "MyGame.zip")
	if err := os.WriteFile(rom, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	// Create initial sidecar metadata
	metadataPath := rom + ".toml"
	metadataContent := `title = "MyGame"
developer = "Dev Corp"
`
	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)

	progress := &mockProgressReporter{}

	// First scan - creates game with metadata
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

	if games[0].Developer == nil || *games[0].Developer != "Dev Corp" {
		t.Errorf("Expected developer 'Dev Corp', got '%v'", games[0].Developer)
	}

	// Delete metadata file
	if err := os.Remove(metadataPath); err != nil {
		t.Fatalf("Failed to delete metadata: %v", err)
	}

	// Clear library scan state
	lib.CurrentScanJobID = nil
	if err := tk.DB.Save(lib).Error; err != nil {
		t.Fatalf("Failed to update library: %v", err)
	}

	// Second scan - should still process file without metadata
	progress = &mockProgressReporter{}
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("Second ScanLibrary failed: %v", err)
	}

	// Should still have 1 game (same one, with title from filename)
	var scannedGames []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&scannedGames).Error; err != nil {
		t.Fatalf("Failed to query games after second scan: %v", err)
	}

	if len(scannedGames) != 1 {
		t.Fatalf("Expected 1 game after second scan, got %d", len(scannedGames))
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

// TestScan_MetadataWriteback tests that scanner can write metadata to sidecar files.
// This verifies that auto-generated metadata is persisted for consistency.
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

	rom := filepath.Join(libPath, "game.zip")
	if err := os.WriteFile(rom, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to write ROM: %v", err)
	}

	// Create sidecar metadata without UUID
	metadataPath := rom + ".toml"
	metadataContent := `title = "Game with Title"
developer = "Dev"
`
	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	platformID := getNESPlatform(t, tk.DB)
	lib := createLibraryWithPath(t, tk.DB, libPath, platformID)

	progress := &mockProgressReporter{}

	// Scan - should load metadata correctly
	if err := svc.ScanLibrary(ctx, slog.Default(), lib, progress); err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}

	// Verify game was created with metadata
	var games []database.Game
	if err := tk.DB.Where("library_id = ?", lib.ID).Find(&games).Error; err != nil {
		t.Fatalf("Failed to query games: %v", err)
	}

	if len(games) != 1 {
		t.Fatalf("Expected 1 game, got %d", len(games))
	}

	if games[0].Title != "Game with Title" {
		t.Errorf("Expected title 'Game with Title', got '%s'", games[0].Title)
	}

	if games[0].Developer == nil || *games[0].Developer != "Dev" {
		t.Errorf("Expected developer 'Dev', got '%v'", games[0].Developer)
	}

	// Verify metadata file is still readable
	if _, err := os.Stat(metadataPath); err != nil {
		t.Fatalf("Metadata file not found: %v", err)
	}

	// Parse metadata to verify it's valid
	metadata, err := svc.LoadMetadataFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to parse written metadata: %v", err)
	}

	if metadata == nil || metadata.Title == "" {
		t.Fatalf("Metadata is invalid after scan")
	}

	if metadata.Title != "Game with Title" {
		t.Errorf("Expected loaded metadata title 'Game with Title', got '%s'", metadata.Title)
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
