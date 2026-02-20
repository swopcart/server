package library

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/jobs"
	"gorm.io/gorm"
)

func (svc *LibraryService) RegisterJobs(jobSvc *jobs.JobService) error {
	errs := []error{
		jobSvc.RegisterScheduledJob(
			svc.context,
			"library.scan",
			"Scan library for added, removed or updated entries",
			"0 0 3 * * *",
			svc.libraryScanJob,
			jobs.WithPriority(10),
			jobs.WithQueue("default"),
			jobs.WithDefaultParameters(jobs.Params{
				"libraries": "*",
			}),
		),
	}

	return errors.Join(errs...)
}

func (svc *LibraryService) libraryScanJob(
	ctx context.Context,
	logger *slog.Logger,
	params jobs.Params,
	progress jobs.ProgressReporter,
) error {
	librariesParam := params["libraries"]

	// Get list of libraries to scan
	var libraries []database.Library
	query := svc.db.WithContext(ctx)

	if librariesParam != "*" {
		// If specific library ID provided, filter by it
		query = query.Where("id = ?", librariesParam)
	}

	if err := query.Find(&libraries).Error; err != nil {
		return err
	}

	for _, lib := range libraries {
		if err := svc.scanLibrary(ctx, logger, &lib, progress); err != nil {
			logger.ErrorContext(ctx, "failed to scan library", "id", lib.ID, "error", err)
			// Continue scanning other libraries
		}
	}

	return nil
}

// scanLibrary scans a single library for games
func (svc *LibraryService) scanLibrary(
	ctx context.Context,
	logger *slog.Logger,
	lib *database.Library,
	progress jobs.ProgressReporter,
) error {
	// Check if scan is already in progress
	if lib.CurrentScanJobID != nil {
		return errors.New("scan already in progress for this library")
	}

	// Mark scan as in progress
	jobID := uuid.New()
	now := time.Now()
	if err := svc.db.WithContext(ctx).Model(lib).Updates(map[string]interface{}{
		"scan_status":         "scanning",
		"current_scan_job_id": jobID,
		"last_scan_error":     nil,
	}).Error; err != nil {
		return err
	}

	defer func() {
		// Clear scan status
		svc.db.WithContext(ctx).Model(lib).Updates(map[string]interface{}{
			"scan_status":         "idle",
			"current_scan_job_id": nil,
			"last_scanned_at":     now,
		})
	}()

	// Get library paths and platform
	paths, err := svc.GetLibraryPaths(ctx, lib.ID)
	if err != nil {
		errMsg := err.Error()
		lib.LastScanError = &errMsg
		return err
	}

	platform, err := svc.GetLibraryPlatform(ctx, lib.ID)
	if err != nil {
		return err
	}

	extensions, err := svc.GetPlatformExtensions(ctx, platform.ID)
	if err != nil {
		return err
	}

	// Track all files found during scan
	foundFiles := make(map[string]bool) // filePath -> true

	// Scan each path
	totalRecords := 0
	processedRecords := 0

	// First pass: count total files
	for _, scanPath := range paths {
		if err := filepath.Walk(scanPath, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				logger.WarnContext(ctx, "error accessing path", "path", filePath, "error", err)
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if matchesExtension(filePath, extensions) {
				totalRecords++
			}
			return nil
		}); err != nil {
			logger.WarnContext(ctx, "error walking directory", "path", scanPath, "error", err)
		}
	}

	progress.UpdateProgress(totalRecords, 0, "")

	// Second pass: process entries in library directories
	// Process ONLY top-level entries (files or folders, not subdirectories)
	for _, scanPath := range paths {
		entries, err := os.ReadDir(scanPath)
		if err != nil {
			logger.WarnContext(ctx, "failed to read directory", "path", scanPath, "error", err)
			continue
		}

		for _, entry := range entries {
			entryPath := filepath.Join(scanPath, entry.Name())

			if entry.IsDir() {
				// Process folder: look for metadata.toml or binaries inside
				if err := svc.processFolderGame(ctx, logger, lib.ID, entryPath, platform, extensions, foundFiles); err != nil {
					logger.WarnContext(ctx, "failed to process folder game", "folder", entryPath, "error", err)
				}
				processedRecords++
				progress.UpdateProgress(totalRecords, processedRecords, entry.Name())
			} else {
				// Process flat file: check for .meta or .toml sidecar
				if matchesExtension(entryPath, extensions) {
					if err := svc.processFlatGame(ctx, logger, lib.ID, entryPath, platform); err != nil {
						logger.WarnContext(ctx, "failed to process flat game", "file", entryPath, "error", err)
					}
					foundFiles[entryPath] = true
					processedRecords++
					progress.UpdateProgress(totalRecords, processedRecords, entry.Name())
				}
			}
		}
	}

	// Soft-delete games that are no longer present
	var allGames []database.Game
	if err := svc.db.WithContext(ctx).Where("library_id = ?", lib.ID).Find(&allGames).Error; err != nil {
		return err
	}

	for _, game := range allGames {
		var versions []database.GameVersion
		svc.db.WithContext(ctx).Where("game_id = ?", game.ID).Find(&versions)

		gameFound := false
		for _, version := range versions {
			if foundFiles[version.FilePath] {
				gameFound = true
				break
			}
		}

		if !gameFound && game.DeletedAt.Time.IsZero() {
			// Game no longer exists on disk, soft-delete it
			if err := svc.db.WithContext(ctx).Delete(&game).Error; err != nil {
				logger.WarnContext(ctx, "failed to soft-delete game", "id", game.ID, "error", err)
			}
		}
	}

	logger.InfoContext(ctx, "Library scan completed", "id", lib.ID, "total", processedRecords)
	return nil
}

// generateMetadataFromFolder scans a folder for ROM files and generates metadata
func (svc *LibraryService) generateMetadataFromFolder(
	ctx context.Context,
	folderPath string,
	platformName string,
	extensions []string,
) (*GameMetadata, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read folder: %w", err)
	}

	// Collect all ROM files (top-level only, not in subdirectories)
	var romFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && matchesExtension(filepath.Join(folderPath, entry.Name()), extensions) {
			romFiles = append(romFiles, entry)
		}
	}

	if len(romFiles) == 0 {
		return nil, nil // No ROMs found
	}

	// Use folder name as game title
	folderName := filepath.Base(folderPath)

	// Generate metadata from the folder name and ROM files
	metadata := &GameMetadata{
		Title:    folderName,
		Platform: platformName,
		Versions: make([]VersionMetadata, 0),
	}

	// Create a version for each ROM file
	for _, romFile := range romFiles {
		filename := romFile.Name()

		// Extract metadata from filename to get regions and other info
		fileMetadata := svc.ExtractMetadataFromFilename(filename, platformName)

		// Use filename (without extension) as version name
		versionName := filename
		if idx := strings.LastIndexByte(versionName, '.'); idx >= 0 {
			versionName = versionName[:idx]
		}

		version := VersionMetadata{
			Name:     versionName,
			Filename: filename,
			Regions:  fileMetadata.Regions,
		}

		metadata.Versions = append(metadata.Versions, version)

		// Inherit developer/publisher from first ROM's extracted metadata
		if len(metadata.Versions) == 1 {
			if fileMetadata.Developer != "" {
				metadata.Developer = fileMetadata.Developer
			}
			if fileMetadata.Publisher != "" {
				metadata.Publisher = fileMetadata.Publisher
			}
		}
	}

	return metadata, nil
}

// processFolderGame processes a game folder with metadata.toml and ROM files inside
func (svc *LibraryService) processFolderGame(
	ctx context.Context,
	logger *slog.Logger,
	libraryID uuid.UUID,
	folderPath string,
	platform *database.Platform,
	extensions []string,
	foundFiles map[string]bool,
) error {
	// Try to load metadata.toml from folder
	metadataPath := filepath.Join(folderPath, "metadata.toml")
	metadata, err := svc.LoadMetadataFromFile(metadataPath)
	if err != nil {
		logger.WarnContext(ctx, "failed to load metadata.toml", "folder", folderPath, "error", err)
		return nil // Skip folder if metadata can't be parsed
	}

	// If no metadata file exists, auto-generate from folder structure
	if metadata == nil {
		metadata, err = svc.generateMetadataFromFolder(ctx, folderPath, platform.Name, extensions)
		if err != nil {
			logger.WarnContext(ctx, "failed to generate metadata from folder", "folder", folderPath, "error", err)
			return nil
		}

		if metadata == nil || len(metadata.Versions) == 0 {
			// No ROMs found in folder
			return nil
		}

		// Save generated metadata for future scans
		if err := svc.SaveMetadataToFile(metadataPath, metadata); err != nil {
			logger.WarnContext(ctx, "failed to save generated metadata", "path", metadataPath, "error", err)
		}
	}

	// Get or create game
	game, err := svc.GetOrCreateGame(ctx, libraryID, folderPath, metadata.Title)
	if err != nil {
		return fmt.Errorf("failed to get/create game: %w", err)
	}

	// Process each version in metadata
	for _, versionMeta := range metadata.Versions {
		filePath := filepath.Join(folderPath, versionMeta.Filename)

		// Verify file exists
		info, err := os.Stat(filePath)
		if err != nil {
			logger.WarnContext(ctx, "version file not found", "file", filePath, "error", err)
			continue
		}

		foundFiles[filePath] = true

		// Calculate hashes
		var hashes *Hashes
		var existingVersion database.GameVersion
		if err := svc.db.WithContext(ctx).Where("file_path = ?", filePath).First(&existingVersion).Error; err == gorm.ErrRecordNotFound {
			// New file, calculate hashes
			h, err := svc.CalculateFileHashes(filePath)
			if err != nil {
				logger.WarnContext(ctx, "failed to calculate hashes", "file", filePath, "error", err)
				hashes = &Hashes{}
			} else {
				hashes = h
			}
		} else if err != nil {
			logger.WarnContext(ctx, "error checking existing version", "file", filePath, "error", err)
			hashes = &Hashes{}
		} else {
			hashes = &Hashes{
				MD5:    existingVersion.MD5,
				SHA1:   existingVersion.SHA1,
				SHA256: existingVersion.SHA256,
				Blake3: existingVersion.Blake3,
			}
		}

		// Use metadata version name
		versionName := versionMeta.Name
		if versionName == "" {
			// Fall back to filename without extension
			versionName = filepath.Base(versionMeta.Filename)
			if idx := strings.LastIndexByte(versionName, '.'); idx >= 0 {
				versionName = versionName[:idx]
			}
		}

		// Create/update version
		_, err = svc.UpsertGameVersion(ctx, game.ID, filePath, versionName, info.Size(), hashes.ToMap())
		if err != nil {
			logger.ErrorContext(ctx, "failed to upsert version", "file", filePath, "error", err)
		}
	}

	// Update game metadata from metadata.toml
	updates := make(map[string]interface{})
	if metadata.Developer != "" {
		updates["developer"] = metadata.Developer
	}
	if metadata.Publisher != "" {
		updates["publisher"] = metadata.Publisher
	}
	if metadata.ReleaseDate != "" {
		updates["released_date"] = metadata.ReleaseDate
	}
	if metadata.Description != "" {
		updates["description"] = metadata.Description
	}

	if len(updates) > 0 {
		if _, err := svc.UpdateGameMetadata(ctx, game.ID, updates); err != nil {
			logger.WarnContext(ctx, "failed to update game metadata", "game", game.ID, "error", err)
		}
	}

	return nil
}

// processFlatGame processes a flat ROM file in a library directory
func (svc *LibraryService) processFlatGame(
	ctx context.Context,
	logger *slog.Logger,
	libraryID uuid.UUID,
	filePath string,
	platform *database.Platform,
) error {
	filename := filepath.Base(filePath)

	// Try to load metadata sidecar (metadata.toml or .meta.toml)
	metadataPath := filePath + ".toml"
	metadata, err := svc.LoadMetadataFromFile(metadataPath)
	if err != nil {
		logger.WarnContext(ctx, "failed to load metadata sidecar", "file", filePath, "error", err)
		// Fall back to generating metadata from filename
		metadata = svc.ExtractMetadataFromFilename(filename, platform.Name)
	}

	if metadata == nil || metadata.Title == "" {
		// Generate metadata from filename
		metadata = svc.ExtractMetadataFromFilename(filename, platform.Name)
	}

	// Get or create game
	game, err := svc.GetOrCreateGame(ctx, libraryID, filepath.Dir(filePath), metadata.Title)
	if err != nil {
		return fmt.Errorf("failed to get/create game: %w", err)
	}

	// Calculate hashes
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	var hashes *Hashes
	var existingVersion database.GameVersion
	if err := svc.db.WithContext(ctx).Where("file_path = ?", filePath).First(&existingVersion).Error; err == gorm.ErrRecordNotFound {
		// New file, calculate hashes
		h, err := svc.CalculateFileHashes(filePath)
		if err != nil {
			logger.WarnContext(ctx, "failed to calculate hashes", "file", filePath, "error", err)
			hashes = &Hashes{}
		} else {
			hashes = h
		}
	} else if err != nil {
		logger.WarnContext(ctx, "error checking existing version", "file", filePath, "error", err)
		hashes = &Hashes{}
	} else {
		hashes = &Hashes{
			MD5:    existingVersion.MD5,
			SHA1:   existingVersion.SHA1,
			SHA256: existingVersion.SHA256,
			Blake3: existingVersion.Blake3,
		}
	}

	// Use metadata version name if available, otherwise use filename without extension
	versionName := ""
	if len(metadata.Versions) > 0 {
		// Get first version's name
		for _, v := range metadata.Versions {
			versionName = v.Name
			break
		}
	}
	if versionName == "" {
		versionName = filename[:len(filename)-len(filepath.Ext(filename))]
	}

	// Create/update version
	_, err = svc.UpsertGameVersion(ctx, game.ID, filePath, versionName, info.Size(), hashes.ToMap())
	if err != nil {
		logger.ErrorContext(ctx, "failed to upsert version", "file", filePath, "error", err)
	}

	// Update game metadata if available
	updates := make(map[string]interface{})
	if metadata.Developer != "" {
		updates["developer"] = metadata.Developer
	}
	if metadata.Publisher != "" {
		updates["publisher"] = metadata.Publisher
	}
	if metadata.ReleaseDate != "" {
		updates["released_date"] = metadata.ReleaseDate
	}
	if metadata.Description != "" {
		updates["description"] = metadata.Description
	}

	if len(updates) > 0 {
		if _, err := svc.UpdateGameMetadata(ctx, game.ID, updates); err != nil {
			logger.WarnContext(ctx, "failed to update game metadata", "game", game.ID, "error", err)
		}
	}

	// Save metadata sidecar if it doesn't exist
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		if err := svc.SaveMetadataToFile(metadataPath, metadata); err != nil {
			logger.WarnContext(ctx, "failed to save metadata sidecar", "path", metadataPath, "error", err)
		}
	}

	return nil
}

// matchesExtension checks if a file matches any of the given extensions
func matchesExtension(filePath string, extensions []string) bool {
	if len(extensions) == 0 {
		return true // No extension filter
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	for _, allowedExt := range extensions {
		if strings.EqualFold(ext, allowedExt) {
			return true
		}
	}
	return false
}
