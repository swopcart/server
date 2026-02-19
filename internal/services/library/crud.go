package library

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"gorm.io/gorm"
)

// CreateLibraryRequest is the request to create a new library
type CreateLibraryRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	PlatformID  uint     `json:"platformId"`
	Paths       []string `json:"paths"`
}

// UpdateLibraryRequest is the request to update a library
type UpdateLibraryRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Paths       []string `json:"paths,omitempty"`
}

// LibraryResponse is the response for library queries
type LibraryResponse struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	PlatformID       uint      `json:"platformId"`
	PlatformName     string    `json:"platformName,omitempty"`
	Paths            []string  `json:"paths"`
	GameCount        int64     `json:"gameCount"`
	LastScannedAt    *string   `json:"lastScannedAt,omitempty"`
	ScanStatus       string    `json:"scanStatus"`
	LastScanError    *string   `json:"lastScanError,omitempty"`
	CurrentScanJobID *string   `json:"currentScanJobId,omitempty"`
}

// CreateLibrary creates a new library with path validation
func (svc *LibraryService) CreateLibrary(ctx context.Context, req CreateLibraryRequest) (*database.Library, error) {
	// Validate platform exists
	var platform database.Platform
	if err := svc.db.WithContext(ctx).First(&platform, req.PlatformID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("platform not found: %w", err)
		}
		return nil, err
	}

	// Validate all paths exist and are directories
	for _, path := range req.Paths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("path does not exist: %s", path)
			}
			return nil, fmt.Errorf("cannot access path %s: %w", path, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("path is not a directory: %s", path)
		}
	}

	// Marshal paths to JSON
	pathsJSON, err := json.Marshal(req.Paths)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal paths: %w", err)
	}
	pathsStr := string(pathsJSON)

	// Create library
	library := &database.Library{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		PlatformID:  req.PlatformID,
		Paths:       &pathsStr,
		ScanStatus:  "idle",
	}

	if err := svc.db.WithContext(ctx).Create(library).Error; err != nil {
		return nil, fmt.Errorf("failed to create library: %w", err)
	}

	svc.logger.InfoContext(ctx, "Library created", "id", library.ID, "name", library.Name, "paths", len(req.Paths))
	return library, nil
}

// ListLibraries returns all libraries with game counts
func (svc *LibraryService) ListLibraries(ctx context.Context) ([]LibraryResponse, error) {
	var libraries []database.Library
	if err := svc.db.WithContext(ctx).Find(&libraries).Error; err != nil {
		return nil, err
	}

	responses := make([]LibraryResponse, len(libraries))
	for i, lib := range libraries {
		// Get game count
		var gameCount int64
		svc.db.WithContext(ctx).Model(&database.Game{}).Where("library_id = ?", lib.ID).Count(&gameCount)

		// Parse paths
		var paths []string
		if lib.Paths != nil {
			if err := json.Unmarshal([]byte(*lib.Paths), &paths); err != nil {
				svc.logger.WarnContext(ctx, "failed to parse paths", "id", lib.ID, "error", err)
				paths = []string{}
			}
		}

		// Get platform name
		var platform database.Platform
		platformName := ""
		if lib.PlatformID > 0 {
			svc.db.WithContext(ctx).First(&platform, lib.PlatformID)
			platformName = platform.Name
		}

		// Format timestamps
		var lastScannedAt *string
		if lib.LastScannedAt != nil {
			ts := lib.LastScannedAt.Format("2006-01-02T15:04:05Z07:00")
			lastScannedAt = &ts
		}

		var currentScanJobID *string
		if lib.CurrentScanJobID != nil {
			id := lib.CurrentScanJobID.String()
			currentScanJobID = &id
		}

		responses[i] = LibraryResponse{
			ID:               lib.ID,
			Name:             lib.Name,
			Description:      lib.Description,
			PlatformID:       lib.PlatformID,
			PlatformName:     platformName,
			Paths:            paths,
			GameCount:        gameCount,
			LastScannedAt:    lastScannedAt,
			ScanStatus:       lib.ScanStatus,
			LastScanError:    lib.LastScanError,
			CurrentScanJobID: currentScanJobID,
		}
	}

	return responses, nil
}

// GetLibrary returns a single library with game count
func (svc *LibraryService) GetLibrary(ctx context.Context, libraryID uuid.UUID) (*LibraryResponse, error) {
	var lib database.Library
	if err := svc.db.WithContext(ctx).First(&lib, libraryID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("library not found")
		}
		return nil, err
	}

	// Get game count
	var gameCount int64
	svc.db.WithContext(ctx).Model(&database.Game{}).Where("library_id = ?", lib.ID).Count(&gameCount)

	// Parse paths
	var paths []string
	if lib.Paths != nil {
		if err := json.Unmarshal([]byte(*lib.Paths), &paths); err != nil {
			svc.logger.WarnContext(ctx, "failed to parse paths", "id", lib.ID, "error", err)
			paths = []string{}
		}
	}

	// Get platform name
	var platform database.Platform
	platformName := ""
	if lib.PlatformID > 0 {
		svc.db.WithContext(ctx).First(&platform, lib.PlatformID)
		platformName = platform.Name
	}

	// Format timestamps
	var lastScannedAt *string
	if lib.LastScannedAt != nil {
		ts := lib.LastScannedAt.Format("2006-01-02T15:04:05Z07:00")
		lastScannedAt = &ts
	}

	var currentScanJobID *string
	if lib.CurrentScanJobID != nil {
		id := lib.CurrentScanJobID.String()
		currentScanJobID = &id
	}

	return &LibraryResponse{
		ID:               lib.ID,
		Name:             lib.Name,
		Description:      lib.Description,
		PlatformID:       lib.PlatformID,
		PlatformName:     platformName,
		Paths:            paths,
		GameCount:        gameCount,
		LastScannedAt:    lastScannedAt,
		ScanStatus:       lib.ScanStatus,
		LastScanError:    lib.LastScanError,
		CurrentScanJobID: currentScanJobID,
	}, nil
}

// UpdateLibrary updates a library (allows path changes)
func (svc *LibraryService) UpdateLibrary(ctx context.Context, libraryID uuid.UUID, req UpdateLibraryRequest) (*database.Library, error) {
	var lib database.Library
	if err := svc.db.WithContext(ctx).First(&lib, libraryID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("library not found")
		}
		return nil, err
	}

	// Validate paths if provided
	if len(req.Paths) > 0 {
		for _, path := range req.Paths {
			info, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, fmt.Errorf("path does not exist: %s", path)
				}
				return nil, fmt.Errorf("cannot access path %s: %w", path, err)
			}
			if !info.IsDir() {
				return nil, fmt.Errorf("path is not a directory: %s", path)
			}
		}

		// Marshal paths to JSON
		pathsJSON, err := json.Marshal(req.Paths)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal paths: %w", err)
		}
		pathsStr := string(pathsJSON)
		lib.Paths = &pathsStr
	}

	if req.Name != nil {
		lib.Name = *req.Name
	}
	if req.Description != nil {
		lib.Description = *req.Description
	}

	if err := svc.db.WithContext(ctx).Save(&lib).Error; err != nil {
		return nil, fmt.Errorf("failed to update library: %w", err)
	}

	svc.logger.InfoContext(ctx, "Library updated", "id", lib.ID, "name", lib.Name)
	return &lib, nil
}

// DeleteLibrary soft-deletes a library
func (svc *LibraryService) DeleteLibrary(ctx context.Context, libraryID uuid.UUID) error {
	if err := svc.db.WithContext(ctx).Delete(&database.Library{}, libraryID).Error; err != nil {
		return fmt.Errorf("failed to delete library: %w", err)
	}

	svc.logger.InfoContext(ctx, "Library deleted", "id", libraryID)
	return nil
}

// GetLibraryPaths returns the parsed paths for a library
func (svc *LibraryService) GetLibraryPaths(ctx context.Context, libraryID uuid.UUID) ([]string, error) {
	var lib database.Library
	if err := svc.db.WithContext(ctx).First(&lib, libraryID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("library not found")
		}
		return nil, err
	}

	var paths []string
	if lib.Paths != nil {
		if err := json.Unmarshal([]byte(*lib.Paths), &paths); err != nil {
			return nil, fmt.Errorf("failed to parse paths: %w", err)
		}
	}
	return paths, nil
}

// GetLibraryPlatform returns the platform for a library
func (svc *LibraryService) GetLibraryPlatform(ctx context.Context, libraryID uuid.UUID) (*database.Platform, error) {
	var lib database.Library
	if err := svc.db.WithContext(ctx).First(&lib, libraryID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("library not found")
		}
		return nil, err
	}

	var platform database.Platform
	if err := svc.db.WithContext(ctx).First(&platform, lib.PlatformID).Error; err != nil {
		return nil, fmt.Errorf("failed to get platform: %w", err)
	}

	return &platform, nil
}

// GetPlatformExtensions returns the extensions for a platform
func (svc *LibraryService) GetPlatformExtensions(ctx context.Context, platformID uint) ([]string, error) {
	var platform database.Platform
	if err := svc.db.WithContext(ctx).First(&platform, platformID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("platform not found")
		}
		return nil, err
	}

	var extensions []string
	if platform.Extensions != nil {
		if err := json.Unmarshal([]byte(*platform.Extensions), &extensions); err != nil {
			return nil, fmt.Errorf("failed to parse extensions: %w", err)
		}
	}
	return extensions, nil
}

// GetOrCreateGame gets an existing game or creates a new one
func (svc *LibraryService) GetOrCreateGame(ctx context.Context, libraryID uuid.UUID, dirPath string, title string) (*database.Game, error) {
	var game database.Game

	// Try to find existing game by library ID and directory path (we'll store this separately if needed)
	// For now, check by title and library
	result := svc.db.WithContext(ctx).
		Where("library_id = ? AND title = ?", libraryID, title).
		First(&game)

	if result.Error == nil {
		// Game already exists
		return &game, nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return nil, result.Error
	}

	// Create new game
	var lib database.Library
	if err := svc.db.WithContext(ctx).First(&lib, libraryID).Error; err != nil {
		return nil, fmt.Errorf("library not found: %w", err)
	}

	game = database.Game{
		ID:         uuid.New(),
		LibraryID:  libraryID,
		PlatformID: lib.PlatformID,
		Title:      title,
	}

	if err := svc.db.WithContext(ctx).Create(&game).Error; err != nil {
		return nil, fmt.Errorf("failed to create game: %w", err)
	}

	svc.logger.DebugContext(ctx, "Game created", "id", game.ID, "title", game.Title)
	return &game, nil
}

// UpsertGameVersion creates or updates a game version
func (svc *LibraryService) UpsertGameVersion(ctx context.Context, gameID uuid.UUID, filePath string, versionName string, fileSize int64, hashes map[string]string) (*database.GameVersion, error) {
	var version database.GameVersion

	// Try to find existing version by file path
	result := svc.db.WithContext(ctx).
		Where("file_path = ?", filePath).
		First(&version)

	if result.Error == nil {
		// Version exists, update it
		version.VersionName = versionName
		version.FileSize = fileSize
		if hash, ok := hashes["md5"]; ok {
			version.MD5 = hash
		}
		if hash, ok := hashes["sha1"]; ok {
			version.SHA1 = hash
		}
		if hash, ok := hashes["sha256"]; ok {
			version.SHA256 = hash
		}
		if hash, ok := hashes["blake3"]; ok {
			version.Blake3 = hash
		}

		if err := svc.db.WithContext(ctx).Save(&version).Error; err != nil {
			return nil, fmt.Errorf("failed to update version: %w", err)
		}

		return &version, nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return nil, result.Error
	}

	// Create new version
	version = database.GameVersion{
		ID:          uuid.New(),
		GameID:      gameID,
		VersionName: versionName,
		FilePath:    filePath,
		FileSize:    fileSize,
	}

	if hash, ok := hashes["md5"]; ok {
		version.MD5 = hash
	}
	if hash, ok := hashes["sha1"]; ok {
		version.SHA1 = hash
	}
	if hash, ok := hashes["sha256"]; ok {
		version.SHA256 = hash
	}
	if hash, ok := hashes["blake3"]; ok {
		version.Blake3 = hash
	}

	if err := svc.db.WithContext(ctx).Create(&version).Error; err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	svc.logger.DebugContext(ctx, "Game version created", "id", version.ID, "file", filepath.Base(filePath))
	return &version, nil
}

// UpdateGameMetadata updates a game's metadata
func (svc *LibraryService) UpdateGameMetadata(ctx context.Context, gameID uuid.UUID, updates map[string]interface{}) (*database.Game, error) {
	var game database.Game
	if err := svc.db.WithContext(ctx).First(&game, gameID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("game not found")
		}
		return nil, err
	}

	// Update fields if provided
	if title, ok := updates["title"].(string); ok {
		game.Title = title
	}
	if developer, ok := updates["developer"].(string); ok {
		game.Developer = &developer
	}
	if publisher, ok := updates["publisher"].(string); ok {
		game.Publisher = &publisher
	}
	if description, ok := updates["description"].(string); ok {
		game.Description = &description
	}

	if err := svc.db.WithContext(ctx).Save(&game).Error; err != nil {
		return nil, fmt.Errorf("failed to update game: %w", err)
	}

	svc.logger.DebugContext(ctx, "Game metadata updated", "id", game.ID)
	return &game, nil
}

// PlatformResponse is the response for platform queries
type PlatformResponse struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Extensions  []string `json:"extensions"`
}

// ListPlatforms returns all available platforms
func (svc *LibraryService) ListPlatforms(ctx context.Context) ([]PlatformResponse, error) {
	var platforms []database.Platform
	if err := svc.db.WithContext(ctx).Find(&platforms).Error; err != nil {
		return nil, err
	}

	responses := make([]PlatformResponse, len(platforms))
	for i, p := range platforms {
		var extensions []string
		if p.Extensions != nil {
			if err := json.Unmarshal([]byte(*p.Extensions), &extensions); err != nil {
				svc.logger.WarnContext(ctx, "failed to parse extensions", "id", p.ID, "error", err)
				extensions = []string{}
			}
		}

		responses[i] = PlatformResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Extensions:  extensions,
		}
	}

	return responses, nil
}
