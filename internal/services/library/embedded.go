package library

import (
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/swopcart/server/internal/config"
)

//go:embed platforms.toml
var defaultPlatformsContent string

// ensureDefaultPlatformsFile creates the platforms.toml file if it doesn't exist
func ensureDefaultPlatformsFile() error {
	dataDir := config.DataDir()

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Create platforms.toml if it doesn't exist
	platformsPath := filepath.Join(dataDir, "platforms.toml")
	if _, err := os.Stat(platformsPath); os.IsNotExist(err) {
		slog.Info("creating default platforms.toml", "path", platformsPath)
		if err := os.WriteFile(platformsPath, []byte(defaultPlatformsContent), 0644); err != nil {
			return err
		}
	}

	return nil
}
