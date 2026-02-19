package library

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"gorm.io/gorm"
)

// GameWithVersions represents a game with all its versions for API responses
type GameWithVersions struct {
	ID           uuid.UUID              `json:"id"`
	Title        string                 `json:"title"`
	PlatformID   uint                   `json:"platformId"`
	Developer    *string                `json:"developer,omitempty"`
	Publisher    *string                `json:"publisher,omitempty"`
	ReleasedDate *string                `json:"releasedDate,omitempty"`
	Description  *string                `json:"description,omitempty"`
	Versions     []GameVersionResponse  `json:"versions"`
	MetadataJSON map[string]interface{} `json:"metadataJson,omitempty"`
}

// GameVersionResponse represents a game version for API responses
type GameVersionResponse struct {
	ID           uuid.UUID              `json:"id"`
	VersionName  string                 `json:"versionName"`
	FilePath     string                 `json:"filePath"`
	FileSize     int64                  `json:"fileSize"`
	MD5          string                 `json:"md5,omitempty"`
	SHA1         string                 `json:"sha1,omitempty"`
	SHA256       string                 `json:"sha256,omitempty"`
	Blake3       string                 `json:"blake3,omitempty"`
	MetadataJSON map[string]interface{} `json:"metadataJson,omitempty"`
}

// GetGamesByLibrary returns all games in a library with pagination
func (svc *LibraryService) GetGamesByLibrary(
	ctx context.Context,
	libraryID uuid.UUID,
	offset int,
	limit int,
	search string,
) ([]GameWithVersions, int64, error) {
	var games []database.Game
	query := svc.db.WithContext(ctx).Where("library_id = ?", libraryID)

	// Apply search filter if provided
	if search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}

	// Get total count
	var total int64
	if err := query.Model(&database.Game{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	if err := query.
		Offset(offset).
		Limit(limit).
		Preload("Versions").
		Find(&games).Error; err != nil {
		return nil, 0, err
	}

	// Convert to response format
	results := make([]GameWithVersions, len(games))
	for i, game := range games {
		results[i] = svc.gameToResponse(game)
	}

	return results, total, nil
}

// GetGameByID returns a single game with all versions
func (svc *LibraryService) GetGameByID(ctx context.Context, gameID uuid.UUID) (*GameWithVersions, error) {
	var game database.Game
	if err := svc.db.WithContext(ctx).
		Where("id = ?", gameID).
		Preload("Versions").
		First(&game).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("game not found")
		}
		return nil, err
	}

	response := svc.gameToResponse(game)
	return &response, nil
}

// GetGameVersionByID returns a specific game version
func (svc *LibraryService) GetGameVersionByID(
	ctx context.Context,
	gameID uuid.UUID,
	versionID uuid.UUID,
) (*database.GameVersion, error) {
	var version database.GameVersion
	if err := svc.db.WithContext(ctx).
		Where("id = ? AND game_id = ?", versionID, gameID).
		First(&version).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("version not found")
		}
		return nil, err
	}

	return &version, nil
}

// gameToResponse converts a database.Game to GameWithVersions response
func (svc *LibraryService) gameToResponse(game database.Game) GameWithVersions {
	// Convert released date to string
	var releasedDate *string
	if game.ReleasedDate != nil {
		dateStr := game.ReleasedDate.Format("2006-01-02")
		releasedDate = &dateStr
	}

	// Convert versions
	versions := make([]GameVersionResponse, len(game.Versions))
	for i, v := range game.Versions {
		versionMeta := make(map[string]interface{})
		if v.MetadataJSON != nil {
			if err := json.Unmarshal([]byte(*v.MetadataJSON), &versionMeta); err != nil {
				svc.logger.Warn("failed to unmarshal version metadata", "error", err)
			}
		}

		versions[i] = GameVersionResponse{
			ID:           v.ID,
			VersionName:  v.VersionName,
			FilePath:     v.FilePath,
			FileSize:     v.FileSize,
			MD5:          v.MD5,
			SHA1:         v.SHA1,
			SHA256:       v.SHA256,
			Blake3:       v.Blake3,
			MetadataJSON: versionMeta,
		}
	}

	// Convert metadata
	gameMeta := make(map[string]interface{})
	if game.Developer != nil && *game.Developer != "" {
		gameMeta["developer"] = *game.Developer
	}
	if game.Publisher != nil && *game.Publisher != "" {
		gameMeta["publisher"] = *game.Publisher
	}
	if game.Description != nil && *game.Description != "" {
		gameMeta["description"] = *game.Description
	}

	return GameWithVersions{
		ID:           game.ID,
		Title:        game.Title,
		PlatformID:   game.PlatformID,
		Developer:    game.Developer,
		Publisher:    game.Publisher,
		ReleasedDate: releasedDate,
		Description:  game.Description,
		Versions:     versions,
		MetadataJSON: gameMeta,
	}
}
