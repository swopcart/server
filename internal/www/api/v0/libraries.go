package v0

import (
	"errors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/services/jobs"
	"github.com/swopcart/server/internal/services/library"
)

// CreateLibraryRequest is the request body for creating a library
type CreateLibraryRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	PlatformID  uint     `json:"platformId" binding:"required"`
	Paths       []string `json:"paths" binding:"required,min=1"`
}

// UpdateLibraryRequest is the request body for updating a library
type UpdateLibraryRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Paths       []string `json:"paths,omitempty"`
}

// TriggerScanRequest is the request body for manually triggering a scan
type TriggerScanRequest struct {
	LibraryID string `json:"libraryId" binding:"required"`
}

// ListLibrariesHandler returns all libraries
func (h *APIHandlers) ListLibrariesHandler(c *gin.Context) {
	// Check admin access
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only administrators can access this resource")
		return
	}

	libraries, err := h.services.Library.ListLibraries(c.Request.Context())
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "failed to list libraries", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": libraries,
		"total": len(libraries),
	})
}

// GetLibraryHandler returns a single library
func (h *APIHandlers) GetLibraryHandler(c *gin.Context) {
	// Check admin access
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only administrators can access this resource")
		return
	}

	libraryID := c.Param("libraryId")
	id, err := uuid.Parse(libraryID)
	if err != nil {
		respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid library ID", ".libraryId")
		return
	}

	lib, err := h.services.Library.GetLibrary(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errors.New("library not found")) {
			respondError(c, http.StatusNotFound, ErrCodeLibraryNotFound, "Library not found")
		} else {
			h.logger.ErrorContext(c.Request.Context(), "failed to get library", "id", id, "error", err)
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusOK, lib)
}

// CreateLibraryHandler creates a new library
func (h *APIHandlers) CreateLibraryHandler(c *gin.Context) {
	// Check admin access
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only administrators can access this resource")
		return
	}

	var req CreateLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	// Validate paths exist
	for _, path := range req.Paths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				respondFieldError(c, http.StatusBadRequest, ErrCodePathNotFound, "Path does not exist: "+path, ".paths")
			} else {
				respondError(c, http.StatusBadRequest, ErrCodeInvalidLibraryPath, "Cannot access path: "+path)
			}
			return
		}
		if !info.IsDir() {
			respondFieldError(c, http.StatusBadRequest, ErrCodePathNotDirectory, "Path is not a directory: "+path, ".paths")
			return
		}
	}

	lib, err := h.services.Library.CreateLibrary(c.Request.Context(), library.CreateLibraryRequest{
		Name:        req.Name,
		Description: req.Description,
		PlatformID:  req.PlatformID,
		Paths:       req.Paths,
	})

	if err != nil {
		if errors.Is(err, errors.New("platform not found")) {
			respondFieldError(c, http.StatusBadRequest, ErrCodePlatformNotFound, "Platform not found", ".platformId")
		} else {
			h.logger.ErrorContext(c.Request.Context(), "failed to create library", "error", err)
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	// Get the full library response with platform info
	fullLib, err := h.services.Library.GetLibrary(c.Request.Context(), lib.ID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "failed to get created library", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, fullLib)
}

// UpdateLibraryHandler updates a library
func (h *APIHandlers) UpdateLibraryHandler(c *gin.Context) {
	// Check admin access
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only administrators can access this resource")
		return
	}

	libraryID := c.Param("libraryId")
	id, err := uuid.Parse(libraryID)
	if err != nil {
		respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid library ID", ".libraryId")
		return
	}

	var req UpdateLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	// Validate paths if provided
	if len(req.Paths) > 0 {
		for _, path := range req.Paths {
			info, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					respondFieldError(c, http.StatusBadRequest, ErrCodePathNotFound, "Path does not exist: "+path, ".paths")
				} else {
					respondError(c, http.StatusBadRequest, ErrCodeInvalidLibraryPath, "Cannot access path: "+path)
				}
				return
			}
			if !info.IsDir() {
				respondFieldError(c, http.StatusBadRequest, ErrCodePathNotDirectory, "Path is not a directory: "+path, ".paths")
				return
			}
		}
	}

	_, err = h.services.Library.UpdateLibrary(c.Request.Context(), id, library.UpdateLibraryRequest{
		Name:        req.Name,
		Description: req.Description,
		Paths:       req.Paths,
	})

	if err != nil {
		if errors.Is(err, errors.New("library not found")) {
			respondError(c, http.StatusNotFound, ErrCodeLibraryNotFound, "Library not found")
		} else {
			h.logger.ErrorContext(c.Request.Context(), "failed to update library", "id", id, "error", err)
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	// Get the full library response with platform info
	fullLib, err := h.services.Library.GetLibrary(c.Request.Context(), id)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "failed to get updated library", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, fullLib)
}

// DeleteLibraryHandler deletes a library
func (h *APIHandlers) DeleteLibraryHandler(c *gin.Context) {
	// Check admin access
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only administrators can access this resource")
		return
	}

	libraryID := c.Param("libraryId")
	id, err := uuid.Parse(libraryID)
	if err != nil {
		respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid library ID", ".libraryId")
		return
	}

	err = h.services.Library.DeleteLibrary(c.Request.Context(), id)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "failed to delete library", "id", id, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TriggerLibraryScanHandler manually triggers a library scan
func (h *APIHandlers) TriggerLibraryScanHandler(c *gin.Context) {
	// Check admin access
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only administrators can access this resource")
		return
	}

	libraryID := c.Param("libraryId")
	id, err := uuid.Parse(libraryID)
	if err != nil {
		respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid library ID", ".libraryId")
		return
	}

	// Check if library exists
	lib, err := h.services.Library.GetLibrary(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusNotFound, ErrCodeLibraryNotFound, "Library not found")
		return
	}

	// Check if scan already in progress
	if lib.CurrentScanJobID != nil {
		respondError(c, http.StatusConflict, ErrCodeScanAlreadyInProgress, "Scan already in progress for this library")
		return
	}

	// Enqueue scan job with parameters
	executionID, err := h.services.Jobs.EnqueueJob(
		c.Request.Context(),
		"library.scan",
		jobs.WithParameters(jobs.Params{"libraries": id.String()}),
	)

	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "failed to enqueue scan job", "id", id, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"executionId": executionID.String(),
		"status":      "pending",
	})
}
