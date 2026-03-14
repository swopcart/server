package library_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/swopcart/server/internal/services/library"
	"github.com/swopcart/server/internal/testkit"
)

func TestLoadMetadataFromFile_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, "test.meta")

	// Create metadata file
	metaContent := `title = "Test Game"
platform = "NES"
developer = "Test Developer"
publisher = "Test Publisher"
release-date = "1990-01-01"
description = "A test game"
regions = ["NTSC", "PAL"]
tags = ["classic", "action"]

[external-ids]
vgdb = "12345"
`

	if err := os.WriteFile(metaPath, []byte(metaContent), 0644); err != nil {
		t.Fatalf("Failed to write metadata file: %v", err)
	}

	metadata, err := librarySvc.LoadMetadataFromFile(metaPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if metadata.Title != "Test Game" {
		t.Errorf("Expected title 'Test Game', got %q", metadata.Title)
	}

	if metadata.Developer != "Test Developer" {
		t.Errorf("Expected developer 'Test Developer', got %q", metadata.Developer)
	}

	if metadata.ExternalIDs["vgdb"] != "12345" {
		t.Errorf("Expected vgdb ID 12345, got %q", metadata.ExternalIDs["vgdb"])
	}

	if len(metadata.Regions) != 2 {
		t.Errorf("Expected 2 regions, got %d", len(metadata.Regions))
	}
}

func TestLoadMetadataFromFile_NotFound(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	metadata, err := librarySvc.LoadMetadataFromFile("/nonexistent/path/metadata.meta")
	if err != nil {
		t.Fatalf("Should not error on missing file: %v", err)
	}

	if metadata != nil {
		t.Error("Expected nil metadata for non-existent file")
	}
}

func TestSaveMetadataToFile_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, "test.meta")

	metadata := &library.GameMetadata{
		Title:       "Test Game",
		Platform:    "NES",
		Developer:   "Test Dev",
		Publisher:   "Test Pub",
		ReleaseDate: "2000-01-15",
		Description: "Test description",
		ExternalIDs: map[string]string{
			"vgdb": "999",
			"igdb": "888",
		},
		Regions: []string{"NTSC", "PAL"},
		Tags:    []string{"action", "adventure"},
	}

	err = librarySvc.SaveMetadataToFile(metaPath, metadata)
	if err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Fatal("Expected metadata file to be created")
	}

	// Load it back and verify
	loaded, err := librarySvc.LoadMetadataFromFile(metaPath)
	if err != nil {
		t.Fatalf("Failed to load saved metadata: %v", err)
	}

	if loaded.Title != metadata.Title {
		t.Errorf("Expected title %q, got %q", metadata.Title, loaded.Title)
	}

	if loaded.ExternalIDs["vgdb"] != "999" {
		t.Errorf("Expected vgdb ID 999, got %q", loaded.ExternalIDs["vgdb"])
	}
}

func TestMetadataToJSON_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	metadata := &library.GameMetadata{
		Title:       "Test Game",
		Developer:   "Test Dev",
		ExternalIDs: map[string]string{"vgdb": "123"},
		Regions:     []string{"NTSC"},
	}

	jsonMeta := librarySvc.MetadataToJSON(metadata)

	if jsonMeta.Title != "Test Game" {
		t.Errorf("Expected title 'Test Game', got %q", jsonMeta.Title)
	}

	if jsonMeta.Developer != "Test Dev" {
		t.Errorf("Expected developer 'Test Dev', got %q", jsonMeta.Developer)
	}

	if jsonMeta.ExternalIDs["vgdb"] != "123" {
		t.Errorf("Expected external ID vgdb=123, got %q", jsonMeta.ExternalIDs["vgdb"])
	}
}

func TestJSONToMetadata_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	jsonMeta := &library.GameMetadataJSON{
		Title:       "Test Game",
		Developer:   "Test Dev",
		ExternalIDs: map[string]string{"vgdb": "123"},
		Regions:     []string{"PAL"},
	}

	metadata := librarySvc.JSONToMetadata(jsonMeta)

	if metadata.Title != "Test Game" {
		t.Errorf("Expected title 'Test Game', got %q", metadata.Title)
	}

	if metadata.ExternalIDs["vgdb"] != "123" {
		t.Errorf("Expected vgdb ID 123, got %q", metadata.ExternalIDs["vgdb"])
	}
}

func TestMetadataToString_Success(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	metadata := &library.GameMetadata{
		Title:     "Test Game",
		Developer: "Test Dev",
	}

	jsonStr, err := librarySvc.MetadataToString(metadata)
	if err != nil {
		t.Fatalf("Failed to convert to string: %v", err)
	}

	// Verify it's valid JSON
	var jsonObj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonObj); err != nil {
		t.Fatalf("Failed to parse as JSON: %v", err)
	}

	if jsonObj["title"] != "Test Game" {
		t.Errorf("Expected title in JSON, got %v", jsonObj["title"])
	}
}

func TestExtractMetadataFromFilename_VGDBId(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	filename := "The Legend of Zelda [vgdb=12345].nes"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "NES")

	if metadata.ExternalIDs["vgdb"] != "12345" {
		t.Errorf("Expected vgdb ID 12345, got %q", metadata.ExternalIDs["vgdb"])
	}

	if metadata.Title != "The Legend of Zelda" {
		t.Errorf("Expected title 'The Legend of Zelda', got %q", metadata.Title)
	}
}

func TestExtractMetadataFromFilename_Regions(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	filename := "Super Mario Bros .pal.nes"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "SNES")

	if len(metadata.Regions) == 0 || metadata.Regions[0] != "PAL" {
		t.Errorf("Expected region PAL, got %v", metadata.Regions)
	}

	if metadata.Title != "Super Mario Bros" {
		t.Errorf("Expected clean title, got %q", metadata.Title)
	}
}

func TestExtractMetadataFromFilename_CountryCodes(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	filename := "Game Title [USA] [EUR].nes"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "NES")

	if len(metadata.Regions) != 2 {
		t.Errorf("Expected 2 regions, got %d: %v", len(metadata.Regions), metadata.Regions)
	}

	if metadata.Title != "Game Title" {
		t.Errorf("Expected clean title 'Game Title', got %q", metadata.Title)
	}
}

func TestExtractMetadataFromFilename_Version(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	filename := "Game Title (v1.0).nes"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "NES")

	if metadata.Title != "Game Title" {
		t.Errorf("Expected clean title 'Game Title', got %q", metadata.Title)
	}
}

func TestExtractMetadataFromFilename_Demo(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	filename := "Game Title (Demo).nes"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "NES")

	if len(metadata.Tags) == 0 || metadata.Tags[0] != "demo" {
		t.Errorf("Expected demo tag, got %v", metadata.Tags)
	}

	if metadata.Title != "Game Title" {
		t.Errorf("Expected clean title 'Game Title', got %q", metadata.Title)
	}
}

func TestExtractMetadataFromFilename_NTSCVariants(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Test .ntsc-u (NTSC USA variant)
	filename := "Super Mario World.ntsc-u.sfc"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "SNES")

	if metadata.Title != "Super Mario World" {
		t.Errorf("Expected clean title 'Super Mario World', got %q", metadata.Title)
	}

	if len(metadata.Regions) == 0 {
		t.Errorf("Expected regions for .ntsc-u, got empty")
	}

	hasNTSC := false
	hasUSA := false
	for _, region := range metadata.Regions {
		if region == "NTSC" {
			hasNTSC = true
		}
		if region == "USA" {
			hasUSA = true
		}
	}

	if !hasNTSC || !hasUSA {
		t.Errorf("Expected NTSC and USA regions for .ntsc-u, got %v", metadata.Regions)
	}
}

func TestExtractMetadataFromFilename_PALRegion(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Test .pal (PAL variant)
	filename := "Super Mario World.pal.smc"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "SNES")

	if metadata.Title != "Super Mario World" {
		t.Errorf("Expected clean title 'Super Mario World', got %q", metadata.Title)
	}

	if len(metadata.Regions) == 0 || metadata.Regions[0] != "PAL" {
		t.Errorf("Expected PAL region, got %v", metadata.Regions)
	}
}

func TestExtractMetadataFromFilename_ExternalIDs(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Test single external ID (igdb)
	filename := "Super Metroid [igdb=super-metroid].sfc"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "SNES")

	if metadata.Title != "Super Metroid" {
		t.Errorf("Expected clean title 'Super Metroid', got %q", metadata.Title)
	}

	if igdbID, ok := metadata.ExternalIDs["igdb"]; !ok || igdbID != "super-metroid" {
		t.Errorf("Expected igdb=super-metroid, got %v", metadata.ExternalIDs)
	}
}

func TestExtractMetadataFromFilename_MultipleExternalIDs(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Test multiple external IDs
	filename := "Super Metroid [igdb=super-metroid] [vgdb=12345].sfc"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "SNES")

	if metadata.Title != "Super Metroid" {
		t.Errorf("Expected clean title 'Super Metroid', got %q", metadata.Title)
	}

	if igdbID, ok := metadata.ExternalIDs["igdb"]; !ok || igdbID != "super-metroid" {
		t.Errorf("Expected igdb=super-metroid, got %v", metadata.ExternalIDs)
	}

	if vgdbID, ok := metadata.ExternalIDs["vgdb"]; !ok || vgdbID != "12345" {
		t.Errorf("Expected vgdb=12345, got %v", metadata.ExternalIDs)
	}
}

func TestExtractMetadataFromFilename_ExternalIDsWithRegions(t *testing.T) {
	tk := testkit.New(t)

	librarySvc, err := library.NewLibraryService(tk.T.Context(), tk.Config, tk.Logger, tk.DB)
	if err != nil {
		t.Fatalf("Failed to create library service: %v", err)
	}

	// Test external IDs combined with region variants
	filename := "Super Metroid [igdb=super-metroid].ntsc.sfc"
	metadata := librarySvc.ExtractMetadataFromFilename(filename, "SNES")

	if metadata.Title != "Super Metroid" {
		t.Errorf("Expected clean title 'Super Metroid', got %q", metadata.Title)
	}

	if igdbID, ok := metadata.ExternalIDs["igdb"]; !ok || igdbID != "super-metroid" {
		t.Errorf("Expected igdb=super-metroid, got %v", metadata.ExternalIDs)
	}

	if len(metadata.Regions) == 0 || metadata.Regions[0] != "NTSC" {
		t.Errorf("Expected NTSC region, got %v", metadata.Regions)
	}
}
