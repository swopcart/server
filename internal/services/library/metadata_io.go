package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// GameMetadata represents the metadata stored in TOML files (kebab-case keys)
type GameMetadata struct {
	Title       string            `toml:"title"`
	Platform    string            `toml:"platform,omitempty"`
	Developer   string            `toml:"developer,omitempty"`
	Publisher   string            `toml:"publisher,omitempty"`
	ReleaseDate string            `toml:"release-date,omitempty"`
	Description string            `toml:"description,omitempty"`
	ExternalIDs map[string]string `toml:"external-ids,omitempty"`
	Regions     []string          `toml:"regions,omitempty"`
	Tags        []string          `toml:"tags,omitempty"`
	Versions    []VersionMetadata `toml:"versions,omitempty"`
}

// VersionMetadata represents version information in TOML
type VersionMetadata struct {
	Name     string   `toml:"name"`
	Filename string   `toml:"filename"`
	Regions  []string `toml:"regions,omitempty"`
}

// GameMetadataJSON represents metadata as JSON (camelCase keys for API responses)
type GameMetadataJSON struct {
	Title       string                `json:"title"`
	Platform    string                `json:"platform,omitempty"`
	Developer   string                `json:"developer,omitempty"`
	Publisher   string                `json:"publisher,omitempty"`
	ReleaseDate string                `json:"releaseDate,omitempty"`
	Description string                `json:"description,omitempty"`
	ExternalIDs map[string]string     `json:"externalIds,omitempty"`
	Regions     []string              `json:"regions,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Versions    []VersionMetadataJSON `json:"versions,omitempty"`
}

// VersionMetadataJSON represents version information as JSON
type VersionMetadataJSON struct {
	Name     string   `json:"name"`
	Filename string   `json:"filename"`
	Regions  []string `json:"regions,omitempty"`
}

// LoadMetadataFromFile reads and parses a metadata TOML file
func (svc *LibraryService) LoadMetadataFromFile(filePath string) (*GameMetadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, return nil (not an error)
		}
		return nil, fmt.Errorf("failed to read metadata file %s: %w", filePath, err)
	}

	var metadata GameMetadata
	if err := toml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata file %s: %w", filePath, err)
	}

	return &metadata, nil
}

// SaveMetadataToFile writes metadata to a TOML file
func (svc *LibraryService) SaveMetadataToFile(filePath string, metadata *GameMetadata) error {
	data, err := toml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file %s: %w", filePath, err)
	}

	return nil
}

// MetadataToJSON converts kebab-case TOML metadata to camelCase JSON
func (svc *LibraryService) MetadataToJSON(metadata *GameMetadata) *GameMetadataJSON {
	if metadata == nil {
		return nil
	}

	versions := make([]VersionMetadataJSON, len(metadata.Versions))
	for i, v := range metadata.Versions {
		versions[i] = VersionMetadataJSON(v)
	}

	return &GameMetadataJSON{
		Title:       metadata.Title,
		Platform:    metadata.Platform,
		Developer:   metadata.Developer,
		Publisher:   metadata.Publisher,
		ReleaseDate: metadata.ReleaseDate,
		Description: metadata.Description,
		ExternalIDs: metadata.ExternalIDs,
		Regions:     metadata.Regions,
		Tags:        metadata.Tags,
		Versions:    versions,
	}
}

// JSONToMetadata converts camelCase JSON to kebab-case TOML metadata
func (svc *LibraryService) JSONToMetadata(jsonMeta *GameMetadataJSON) *GameMetadata {
	if jsonMeta == nil {
		return nil
	}

	versions := make([]VersionMetadata, len(jsonMeta.Versions))
	for i, v := range jsonMeta.Versions {
		versions[i] = VersionMetadata(v)
	}

	return &GameMetadata{
		Title:       jsonMeta.Title,
		Platform:    jsonMeta.Platform,
		Developer:   jsonMeta.Developer,
		Publisher:   jsonMeta.Publisher,
		ReleaseDate: jsonMeta.ReleaseDate,
		Description: jsonMeta.Description,
		ExternalIDs: jsonMeta.ExternalIDs,
		Regions:     jsonMeta.Regions,
		Tags:        jsonMeta.Tags,
		Versions:    versions,
	}
}

// MetadataToString serializes metadata to JSON string for storage in database
func (svc *LibraryService) MetadataToString(metadata *GameMetadata) (string, error) {
	jsonMeta := svc.MetadataToJSON(metadata)
	data, err := json.Marshal(jsonMeta)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}
	return string(data), nil
}

// StringToMetadata deserializes metadata from JSON string
func (svc *LibraryService) StringToMetadata(metadataStr string) (*GameMetadata, error) {
	var jsonMeta GameMetadataJSON
	if err := json.Unmarshal([]byte(metadataStr), &jsonMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata from JSON: %w", err)
	}
	return svc.JSONToMetadata(&jsonMeta), nil
}

// ExtractMetadataFromFilename parses filename patterns and extracts metadata
// Patterns:
// - [provider=id] → external ID (e.g., [igdb=super-metroid], [vgdb=12345])
// - .pal, .ntsc → region
// - (v1.0), (Rev A) → version name
// - (Demo), (Beta) → tags
func (svc *LibraryService) ExtractMetadataFromFilename(filename string, platform string) *GameMetadata {
	metadata := &GameMetadata{
		Platform:    platform,
		ExternalIDs: make(map[string]string),
		Regions:     []string{},
		Tags:        []string{},
	}

	// Remove file extension
	title := filename
	if idx := strings.LastIndexByte(title, '.'); idx >= 0 {
		title = title[:idx]
	}

	// Extract external IDs: [provider=id] (e.g., [igdb=super-metroid], [vgdb=12345])
	// Use a loop to handle multiple external IDs
	for {
		bracketStart := strings.Index(title, "[")
		if bracketStart < 0 {
			break
		}

		bracketEnd := strings.Index(title[bracketStart:], "]")
		if bracketEnd < 0 {
			break
		}

		bracketEnd += bracketStart

		// Extract content between brackets
		content := title[bracketStart+1 : bracketEnd]

		// Check if it matches provider=id pattern
		eqIdx := strings.Index(content, "=")
		if eqIdx > 0 {
			provider := content[:eqIdx]
			id := content[eqIdx+1:]

			// Store external ID
			if provider != "" && id != "" {
				metadata.ExternalIDs[provider] = id
			}

			// Remove the external ID from title
			title = title[:bracketStart] + title[bracketEnd+1:]
		} else {
			// Not a provider=id pattern, skip this bracket
			break
		}
	}

	// Extract region: .pal, .ntsc, .ntsc-u, .ntsc-j, etc.
	// Handle hyphenated variants first (.ntsc-u, .ntsc-j) before single (.ntsc)
	if strings.Contains(title, ".ntsc-u") {
		if !contains(metadata.Regions, "NTSC") {
			metadata.Regions = append(metadata.Regions, "NTSC")
		}
		if !contains(metadata.Regions, "USA") {
			metadata.Regions = append(metadata.Regions, "USA")
		}
		title = strings.ReplaceAll(title, ".ntsc-u", "")
	}
	if strings.Contains(title, ".ntsc-j") {
		if !contains(metadata.Regions, "NTSC") {
			metadata.Regions = append(metadata.Regions, "NTSC")
		}
		if !contains(metadata.Regions, "JPN") {
			metadata.Regions = append(metadata.Regions, "JPN")
		}
		title = strings.ReplaceAll(title, ".ntsc-j", "")
	}
	if strings.Contains(title, ".pal") {
		if !contains(metadata.Regions, "PAL") {
			metadata.Regions = append(metadata.Regions, "PAL")
		}
		title = strings.ReplaceAll(title, ".pal", "")
	}
	if strings.Contains(title, ".ntsc") {
		if !contains(metadata.Regions, "NTSC") {
			metadata.Regions = append(metadata.Regions, "NTSC")
		}
		title = strings.ReplaceAll(title, ".ntsc", "")
	}

	// Extract version/tag: (v1.0), (Rev A), (Demo), (Beta)
	if versionMatch := strings.Index(title, "("); versionMatch >= 0 {
		endIdx := strings.Index(title[versionMatch:], ")")
		if endIdx >= 0 {
			versionStr := title[versionMatch+1 : versionMatch+endIdx]

			// Check for demo/beta tags
			if strings.EqualFold(versionStr, "demo") || strings.EqualFold(versionStr, "beta") {
				metadata.Tags = append(metadata.Tags, strings.ToLower(versionStr))
			}
			// Note: Version name is stored in GameVersion.VersionName, not in metadata

			title = title[:versionMatch] + title[versionMatch+endIdx+1:]
		}
	}

	// Extract country codes: [USA], [EUR], [JPN]
	regions := []string{"USA", "EUR", "JPN", "AUS", "CAN"}
	for _, region := range regions {
		pattern := "[" + region + "]"
		if strings.Contains(title, pattern) {
			if !contains(metadata.Regions, region) {
				metadata.Regions = append(metadata.Regions, region)
			}
			title = strings.ReplaceAll(title, pattern, "")
		}
	}

	// Clean up title: remove extra spaces
	title = strings.TrimSpace(title)
	title = strings.Join(strings.Fields(title), " ")
	metadata.Title = title

	return metadata
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ParseReleaseDate parses a release date string in YYYY-MM-DD format
func (svc *LibraryService) ParseReleaseDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		svc.logger.Warn("failed to parse release date", "date", dateStr, "error", err)
		return nil
	}

	return &t
}

// FormatReleaseDate formats a time.Time as YYYY-MM-DD string
func (svc *LibraryService) FormatReleaseDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
