package v0

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UpdateGameMetadataRequest is the request body for updating game metadata
type UpdateGameMetadataRequest struct {
	Title       *string           `json:"title,omitempty"`
	Developer   *string           `json:"developer,omitempty"`
	Publisher   *string           `json:"publisher,omitempty"`
	Description *string           `json:"description,omitempty"`
	ExternalIDs map[string]string `json:"externalIds,omitempty"`
	Regions     []string          `json:"regions,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// GameListResponse is the response for listing games in a library
type GameListResponse struct {
	Items  interface{} `json:"items"`
	Offset int         `json:"offset"`
	Total  int64       `json:"total"`
}

// ListGamesHandler returns games in a library with pagination
func (h *APIHandlers) ListGamesHandler(c *gin.Context) {
	libraryID := c.Param("libraryId")
	_, err := uuid.Parse(libraryID)
	if err != nil {
		respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid library ID", ".libraryId")
		return
	}

	// Get pagination parameters
	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			_ = l // Limit parsed but not yet used
		}
	}

	// Get search query (parsed but not yet used)
	_ = c.Query("search")

	// TODO: Implement game listing with search and pagination
	// For now, return empty list
	c.JSON(http.StatusOK, GameListResponse{
		Items:  []interface{}{},
		Offset: offset,
		Total:  0,
	})
}

// GetGameHandler returns a single game with all versions and metadata
func (h *APIHandlers) GetGameHandler(c *gin.Context) {
	gameID := c.Param("gameId")
	_, err := uuid.Parse(gameID)
	if err != nil {
		respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid game ID", ".gameId")
		return
	}

	// TODO: Implement game fetching with versions
	respondError(c, http.StatusNotImplemented, "NotImplemented", "Game fetching not yet implemented")
}

// DownloadGameVersionHandler streams a game file for download
func (h *APIHandlers) DownloadGameVersionHandler(c *gin.Context) {
	gameID := c.Param("gameId")
	versionID := c.Param("versionId")

	if _, err := uuid.Parse(gameID); err != nil {
		respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid game ID", ".gameId")
		return
	}

	if _, err := uuid.Parse(versionID); err != nil {
		respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid version ID", ".versionId")
		return
	}

	// TODO: Implement game file download with streaming and range request support
	respondError(c, http.StatusNotImplemented, "NotImplemented", "Game download not yet implemented")
}

// UpdateGameMetadataHandler updates game metadata (admin-only)
func (h *APIHandlers) UpdateGameMetadataHandler(c *gin.Context) {
	// Check admin access
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only administrators can access this resource")
		return
	}

	gameID := c.Param("gameId")
	id, err := uuid.Parse(gameID)
	if err != nil {
		respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid game ID", ".gameId")
		return
	}

	var req UpdateGameMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Developer != nil {
		updates["developer"] = *req.Developer
	}
	if req.Publisher != nil {
		updates["publisher"] = *req.Publisher
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	// Update game metadata
	game, err := h.services.Library.UpdateGameMetadata(c.Request.Context(), id, updates)
	if err != nil {
		if errors.Is(err, errors.New("game not found")) {
			respondError(c, http.StatusNotFound, ErrCodeGameNotFound, "Game not found")
		} else {
			h.logger.ErrorContext(c.Request.Context(), "failed to update game metadata", "id", id, "error", err)
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusOK, game)
}

// downloadGameFile handles the actual file streaming with range request support
// This is a helper function for DownloadGameVersionHandler (currently unused, will be used when implementation is complete)
//
//nolint:unused
func (h *APIHandlers) downloadGameFile(c *gin.Context, filePath string, filename string) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(c, http.StatusNotFound, ErrCodeCannotDownloadGame, "Game file not found")
		} else {
			respondError(c, http.StatusInternalServerError, ErrCodeCannotDownloadGame, "Cannot read game file")
		}
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			h.logger.Warn("failed to close file", "path", filePath, "error", err)
		}
	}()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		respondError(c, http.StatusInternalServerError, ErrCodeCannotDownloadGame, "Cannot read game file")
		return err
	}

	fileSize := fileInfo.Size()

	// Set response headers
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Length", strconv.FormatInt(fileSize, 10))

	// Check for Range header (resumable downloads)
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		// Parse range header and handle partial content
		// Format: bytes=start-end
		// For now, we'll implement simple range support
		c.Header("Accept-Ranges", "bytes")
		// TODO: Implement full range request support
	}

	// Stream the file
	if _, err := io.Copy(c.Writer, file); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "error streaming file", "path", filePath, "error", err)
		return err
	}

	return nil
}
