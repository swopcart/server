package library

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
	"gorm.io/gorm"
)

type LibraryService struct {
	context context.Context
	config  *config.Config
	logger  *slog.Logger
	db      *gorm.DB
}

func NewLibraryService(
	ctx context.Context,
	cfg *config.Config,
	l *slog.Logger,
	db *gorm.DB,
) (*LibraryService, error) {
	svc := &LibraryService{
		context: ctx,
		config:  cfg,
		logger:  l,
		db:      db,
	}

	// Load platforms from platforms.toml
	if err := svc.loadPlatforms(ctx); err != nil {
		return nil, fmt.Errorf("failed to load platforms: %w", err)
	}

	return svc, nil
}

// loadPlatforms loads platform definitions from platforms.toml
func (svc *LibraryService) loadPlatforms(ctx context.Context) error {
	dataDir := config.DataDir()
	platformsPath := filepath.Join(dataDir, "platforms.toml")

	// Read platforms.toml
	data, err := os.ReadFile(platformsPath)
	if err != nil {
		return fmt.Errorf("cannot read platforms.toml: %w", err)
	}

	// Parse TOML
	type PlatformEntry struct {
		Name        string   `toml:"name"`
		Description string   `toml:"description"`
		Extensions  []string `toml:"extensions"`
	}

	type PlatformsFile struct {
		Platforms []PlatformEntry `toml:"platforms"`
	}

	var pf PlatformsFile
	if err := toml.Unmarshal(data, &pf); err != nil {
		return fmt.Errorf("failed to parse platforms.toml: %w", err)
	}

	// Upsert platforms into database
	for _, entry := range pf.Platforms {
		// Convert extensions array to JSON string
		extensionsJSON, err := json.Marshal(entry.Extensions)
		if err != nil {
			return fmt.Errorf("failed to marshal extensions for %s: %w", entry.Name, err)
		}
		extensionsStr := string(extensionsJSON)

		// Upsert: create if not exists, update if exists
		platform := &database.Platform{
			Name:        entry.Name,
			Description: entry.Description,
			Extensions:  &extensionsStr,
		}

		result := svc.db.WithContext(ctx).
			Where("name = ?", entry.Name).
			Assign(platform).
			FirstOrCreate(platform)

		if result.Error != nil {
			return fmt.Errorf("failed to upsert platform %s: %w", entry.Name, result.Error)
		}

		svc.logger.DebugContext(ctx, "Platform loaded", "name", entry.Name, "extensions", entry.Extensions)
	}

	svc.logger.InfoContext(ctx, "Platforms loaded successfully", "count", len(pf.Platforms))
	return nil
}
