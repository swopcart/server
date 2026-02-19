package library

import (
	"context"
	"errors"
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

	// Second pass: process files
	for _, scanPath := range paths {
		if err := filepath.Walk(scanPath, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				logger.WarnContext(ctx, "error accessing path", "path", filePath, "error", err)
				return nil
			}

			if info.IsDir() {
				return nil
			}

			// Check if file matches platform extensions
			if !matchesExtension(filePath, extensions) {
				return nil
			}

			foundFiles[filePath] = true

			// Get directory name as game title
			dirPath := filepath.Dir(filePath)
			dirName := filepath.Base(dirPath)

			// Extract metadata from filename
			filename := filepath.Base(filePath)
			metadata := svc.ExtractMetadataFromFilename(filename, platform.Name)

			// Use extracted title if available, otherwise use directory name
			if metadata.Title == "" {
				metadata.Title = dirName
			}

			// Get or create game
			game, err := svc.GetOrCreateGame(ctx, lib.ID, dirPath, metadata.Title)
			if err != nil {
				logger.ErrorContext(ctx, "failed to get/create game", "title", metadata.Title, "error", err)
				return nil
			}

			// Calculate hashes for first scan only
			var hashes *Hashes
			var existingVersion database.GameVersion
			if err := svc.db.WithContext(ctx).Where("file_path = ?", filePath).First(&existingVersion).Error; err == gorm.ErrRecordNotFound {
				// New file, calculate hashes
				hashes, err = svc.CalculateFileHashes(filePath)
				if err != nil {
					logger.WarnContext(ctx, "failed to calculate hashes", "file", filePath, "error", err)
					hashes = &Hashes{} // Use empty hashes
				}
			} else if err != nil {
				logger.WarnContext(ctx, "error checking existing version", "file", filePath, "error", err)
				hashes = &Hashes{}
			} else {
				// File already exists in DB, use existing hashes
				hashes = &Hashes{
					MD5:    existingVersion.MD5,
					SHA1:   existingVersion.SHA1,
					SHA256: existingVersion.SHA256,
					Blake3: existingVersion.Blake3,
				}
			}

			// Create/update game version
			_, err = svc.UpsertGameVersion(ctx, game.ID, filePath, dirName, info.Size(), hashes.ToMap())
			if err != nil {
				logger.ErrorContext(ctx, "failed to upsert version", "file", filePath, "error", err)
			}

			// Save metadata to file if it doesn't exist
			metadataPath := filePath + ".meta"
			if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
				if err := svc.SaveMetadataToFile(metadataPath, metadata); err != nil {
					logger.WarnContext(ctx, "failed to save metadata file", "path", metadataPath, "error", err)
				}
			}

			// Update game metadata if we extracted anything useful
			if metadata.Developer != "" || metadata.Publisher != "" {
				updates := make(map[string]interface{})
				if metadata.Developer != "" {
					updates["developer"] = metadata.Developer
				}
				if metadata.Publisher != "" {
					updates["publisher"] = metadata.Publisher
				}
				if _, err := svc.UpdateGameMetadata(ctx, game.ID, updates); err != nil {
					logger.WarnContext(ctx, "failed to update game metadata", "game", game.ID, "error", err)
				}
			}

			processedRecords++
			progress.UpdateProgress(totalRecords, processedRecords, filename)

			return nil
		}); err != nil {
			logger.WarnContext(ctx, "error walking directory", "path", scanPath, "error", err)
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
