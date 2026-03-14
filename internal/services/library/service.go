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
	"gorm.io/gorm/clause"
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
	// Ensure default platforms file exists
	if err := ensureDefaultPlatformsFile(); err != nil {
		return nil, fmt.Errorf("failed to ensure default platforms file: %w", err)
	}

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
		// If file doesn't exist in test/dev environment, seed default platforms
		if os.IsNotExist(err) {
			svc.logger.WarnContext(ctx, "platforms.toml not found, seeding default platforms", "path", platformsPath)
			return svc.seedDefaultPlatforms(ctx)
		}
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

		platform := &database.Platform{
			Name:        entry.Name,
			Description: entry.Description,
			Extensions:  &extensionsStr,
		}

		result := svc.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{"description", "extensions"}),
			}).
			Create(platform)

		if result.Error != nil {
			return fmt.Errorf("failed to upsert platform %s: %w", entry.Name, result.Error)
		}

		svc.logger.DebugContext(ctx, "Platform loaded", "name", entry.Name, "extensions", entry.Extensions)
	}

	svc.logger.InfoContext(ctx, "Platforms loaded successfully", "count", len(pf.Platforms))
	return nil
}

// seedDefaultPlatforms seeds the database with default platforms for development/testing
func (svc *LibraryService) seedDefaultPlatforms(ctx context.Context) error {
	defaultPlatforms := []struct {
		name        string
		description string
		extensions  []string
	}{
		{"NES", "Nintendo Entertainment System", []string{".nes", ".rom", ".zip"}},
		{"SNES", "Super Nintendo Entertainment System", []string{".sfc", ".smc", ".rom", ".zip"}},
		{"N64", "Nintendo 64", []string{".z64", ".n64", ".rom", ".zip"}},
		{"Game Boy", "Nintendo Game Boy", []string{".gb", ".rom", ".zip"}},
		{"Game Boy Color", "Nintendo Game Boy Color", []string{".gbc", ".rom", ".zip"}},
		{"Game Boy Advance", "Nintendo Game Boy Advance", []string{".gba", ".rom", ".zip"}},
		{"Sega Genesis", "Sega Genesis / Mega Drive", []string{".gen", ".md", ".rom", ".zip"}},
		{"Sega Game Gear", "Sega Game Gear", []string{".gg", ".rom", ".zip"}},
		{"Sega Master System", "Sega Master System", []string{".sms", ".rom", ".zip"}},
		{"Atari 2600", "Atari 2600", []string{".a26", ".bin", ".rom", ".zip"}},
		{"PlayStation 1", "Sony PlayStation 1", []string{".iso", ".cue", ".bin", ".zip"}},
		{"PlayStation 2", "Sony PlayStation 2", []string{".iso", ".bin", ".cue", ".zip"}},
		{"Dreamcast", "Sega Dreamcast", []string{".iso", ".cdi", ".gdi", ".zip"}},
		{"DOS", "MS-DOS", []string{".exe", ".com", ".bat", ".zip"}},
		{"Windows", "Windows PC", []string{".exe", ".iso", ".zip"}},
	}

	for _, p := range defaultPlatforms {
		extensionsJSON, err := json.Marshal(p.extensions)
		if err != nil {
			return fmt.Errorf("failed to marshal extensions for %s: %w", p.name, err)
		}
		extensionsStr := string(extensionsJSON)

		platform := &database.Platform{
			Name:        p.name,
			Description: p.description,
			Extensions:  &extensionsStr,
		}

		result := svc.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{"description", "extensions"}),
			}).
			Create(platform)

		if result.Error != nil {
			return fmt.Errorf("failed to seed platform %s: %w", p.name, result.Error)
		}
	}

	svc.logger.InfoContext(ctx, "Default platforms seeded successfully", "count", len(defaultPlatforms))
	return nil
}
