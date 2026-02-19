package library_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/library"
	"github.com/swopcart/server/internal/testkit"
)

func TestGetGamesByLibrary_Empty(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Create a library first
	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected platform to exist: %v", err)
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

	games, total, err := librarySvc.GetGamesByLibrary(tk.T.Context(), lib.ID, 0, 50, "")
	if err != nil {
		t.Fatalf("Failed to get games: %v", err)
	}

	if len(games) != 0 {
		t.Errorf("Expected 0 games, got %d", len(games))
	}

	if total != 0 {
		t.Errorf("Expected total 0, got %d", total)
	}
}

func TestGetGamesByLibrary_WithGames(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Create library and games
	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected platform to exist: %v", err)
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

	// Create test games
	for i := 0; i < 3; i++ {
		game := &database.Game{
			ID:         uuid.New(),
			LibraryID:  lib.ID,
			PlatformID: platform.ID,
			Title:      "Game " + string(rune(i+49)),
		}
		if err := tk.DB.Create(game).Error; err != nil {
			t.Fatalf("Failed to create game: %v", err)
		}
	}

	games, total, err := librarySvc.GetGamesByLibrary(tk.T.Context(), lib.ID, 0, 50, "")
	if err != nil {
		t.Fatalf("Failed to get games: %v", err)
	}

	if len(games) != 3 {
		t.Errorf("Expected 3 games, got %d", len(games))
	}

	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}
}

func TestGetGamesByLibrary_Pagination(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Create library
	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected platform to exist: %v", err)
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

	// Create 5 games
	for i := 0; i < 5; i++ {
		game := &database.Game{
			ID:         uuid.New(),
			LibraryID:  lib.ID,
			PlatformID: platform.ID,
			Title:      "Game " + string(rune(i+49)),
		}
		if err := tk.DB.Create(game).Error; err != nil {
			t.Fatalf("Failed to create game: %v", err)
		}
	}

	// Get first 2
	games, total, err := librarySvc.GetGamesByLibrary(tk.T.Context(), lib.ID, 0, 2, "")
	if err != nil {
		t.Fatalf("Failed to get games: %v", err)
	}

	if len(games) != 2 {
		t.Errorf("Expected 2 games, got %d", len(games))
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	// Get next 2
	games2, total2, err := librarySvc.GetGamesByLibrary(tk.T.Context(), lib.ID, 2, 2, "")
	if err != nil {
		t.Fatalf("Failed to get games: %v", err)
	}

	if len(games2) != 2 {
		t.Errorf("Expected 2 games, got %d", len(games2))
	}

	// Verify total is still 5
	if total2 != 5 {
		t.Errorf("Expected total 5, got %d", total2)
	}
}

func TestGetGamesByLibrary_Search(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Create library
	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected platform to exist: %v", err)
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

	// Create games with different titles
	titles := []string{"The Legend of Zelda", "Super Mario Bros", "Donkey Kong"}
	for _, title := range titles {
		game := &database.Game{
			ID:         uuid.New(),
			LibraryID:  lib.ID,
			PlatformID: platform.ID,
			Title:      title,
		}
		if err := tk.DB.Create(game).Error; err != nil {
			t.Fatalf("Failed to create game: %v", err)
		}
	}

	// Search for "Zelda"
	games, total, err := librarySvc.GetGamesByLibrary(tk.T.Context(), lib.ID, 0, 50, "Zelda")
	if err != nil {
		t.Fatalf("Failed to get games: %v", err)
	}

	if len(games) != 1 {
		t.Errorf("Expected 1 game matching 'Zelda', got %d", len(games))
	}

	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}

	if games[0].Title != "The Legend of Zelda" {
		t.Errorf("Expected 'The Legend of Zelda', got %q", games[0].Title)
	}

	// Search for "Mario" (case-insensitive)
	games2, total2, err := librarySvc.GetGamesByLibrary(tk.T.Context(), lib.ID, 0, 50, "mario")
	if err != nil {
		t.Fatalf("Failed to get games: %v", err)
	}

	if len(games2) != 1 {
		t.Errorf("Expected 1 game matching 'mario', got %d", len(games2))
	}

	// Verify total is 1
	if total2 != 1 {
		t.Errorf("Expected total 1, got %d", total2)
	}
}

func TestGetGameByID_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Create library and game
	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected platform to exist: %v", err)
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

	gameID := uuid.New()
	game := &database.Game{
		ID:         gameID,
		LibraryID:  lib.ID,
		PlatformID: platform.ID,
		Title:      "Test Game",
		Developer:  strPtr("Test Dev"),
	}
	if err := tk.DB.Create(game).Error; err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Get game
	retrieved, err := librarySvc.GetGameByID(tk.T.Context(), gameID)
	if err != nil {
		t.Fatalf("Failed to get game: %v", err)
	}

	if retrieved.ID != gameID {
		t.Errorf("Expected ID %v, got %v", gameID, retrieved.ID)
	}

	if retrieved.Title != "Test Game" {
		t.Errorf("Expected title 'Test Game', got %q", retrieved.Title)
	}

	if retrieved.Developer == nil || *retrieved.Developer != "Test Dev" {
		t.Errorf("Expected developer 'Test Dev', got %v", retrieved.Developer)
	}
}

func TestGetGameByID_WithVersions(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Create library and game
	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected platform to exist: %v", err)
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

	gameID := uuid.New()
	game := &database.Game{
		ID:         gameID,
		LibraryID:  lib.ID,
		PlatformID: platform.ID,
		Title:      "Test Game",
	}
	if err := tk.DB.Create(game).Error; err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Create 2 versions
	for i := 0; i < 2; i++ {
		version := &database.GameVersion{
			ID:          uuid.New(),
			GameID:      gameID,
			VersionName: "v1." + string(rune(i+48)),
			FilePath:    "/tmp/test" + string(rune(i+48)) + ".rom",
			FileSize:    1024,
		}
		if err := tk.DB.Create(version).Error; err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}
	}

	// Get game
	retrieved, err := librarySvc.GetGameByID(tk.T.Context(), gameID)
	if err != nil {
		t.Fatalf("Failed to get game: %v", err)
	}

	if len(retrieved.Versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(retrieved.Versions))
	}
}

func TestGetGameByID_NotFound(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	nonExistentID := uuid.New()
	_, err = librarySvc.GetGameByID(tk.T.Context(), nonExistentID)
	if err == nil {
		t.Fatal("Expected error for non-existent game")
	}
}

func TestGetGameVersionByID_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Create library and game
	var platform database.Platform
	if err := tk.DB.First(&platform).Error; err != nil {
		t.Fatalf("Expected platform to exist: %v", err)
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

	gameID := uuid.New()
	game := &database.Game{
		ID:         gameID,
		LibraryID:  lib.ID,
		PlatformID: platform.ID,
		Title:      "Test Game",
	}
	if err := tk.DB.Create(game).Error; err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	versionID := uuid.New()
	version := &database.GameVersion{
		ID:          versionID,
		GameID:      gameID,
		VersionName: "1.0",
		FilePath:    "/tmp/test.rom",
		FileSize:    1024,
	}
	if err := tk.DB.Create(version).Error; err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	// Get version
	retrieved, err := librarySvc.GetGameVersionByID(tk.T.Context(), gameID, versionID)
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}

	if retrieved.ID != versionID {
		t.Errorf("Expected ID %v, got %v", versionID, retrieved.ID)
	}

	if retrieved.FilePath != "/tmp/test.rom" {
		t.Errorf("Expected path '/tmp/test.rom', got %q", retrieved.FilePath)
	}
}

func TestGetGameVersionByID_NotFound(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	gameID := uuid.New()
	versionID := uuid.New()
	_, err = librarySvc.GetGameVersionByID(tk.T.Context(), gameID, versionID)
	if err == nil {
		t.Fatal("Expected error for non-existent version")
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}
