package v0

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/services/jobs"
)

// jobsList returns all jobs
func (h *APIHandlers) jobsList(c *gin.Context) {
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only admins can view jobs")
		return
	}

	ctx := c.Request.Context()

	var jobs []struct {
		UUID              uuid.UUID `json:"uuid"`
		Name              string    `json:"name"`
		Description       string    `json:"description"`
		Schedule          *string   `json:"schedule"`
		Enabled           bool      `json:"enabled"`
		Priority          int       `json:"priority"`
		Queue             string    `json:"queue"`
		DefaultParameters *string   `json:"defaultParameters"`
		LastRunAt         *string   `json:"lastRunAt"`
		NextRunAt         *string   `json:"nextRunAt"`
	}

	if err := h.services.Jobs.DB().WithContext(ctx).
		Table("jobs").
		Find(&jobs).Error; err != nil {
		h.getLogger(c).Error("Failed to list jobs", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// jobsGetDetails returns details for a specific job
func (h *APIHandlers) jobsGetDetails(c *gin.Context) {
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only admins can view jobs")
		return
	}

	jobName := c.Param("job_name")
	ctx := c.Request.Context()

	var job struct {
		UUID              uuid.UUID `json:"uuid"`
		Name              string    `json:"name"`
		Description       string    `json:"description"`
		Schedule          *string   `json:"schedule"`
		Enabled           bool      `json:"enabled"`
		Priority          int       `json:"priority"`
		Queue             string    `json:"queue"`
		DefaultParameters *string   `json:"defaultParameters"`
		LastRunAt         *string   `json:"lastRunAt"`
		NextRunAt         *string   `json:"nextRunAt"`
	}

	if err := h.services.Jobs.DB().WithContext(ctx).
		Table("jobs").
		Where("name = ?", jobName).
		First(&job).Error; err != nil {
		respondError(c, http.StatusNotFound, ErrCodeJobNotFound, "Job not found")
		return
	}

	c.JSON(http.StatusOK, job)
}

// jobsGetHistory returns execution history for a job
func (h *APIHandlers) jobsGetHistory(c *gin.Context) {
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only admins can view job history")
		return
	}

	jobName := c.Param("job_name")
	ctx := c.Request.Context()

	// Parse limit from query params
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50
	}

	executions, err := h.services.Jobs.GetJobExecutionHistory(ctx, jobName, limit)
	if err != nil {
		mapError := mapJobsError(err)
		if mapError.code != "" {
			respondError(c, http.StatusNotFound, mapError.code, mapError.message)
			return
		}
		h.getLogger(c).Error("Failed to get job execution history", "err", err, "job", jobName)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, executions)
}

// jobsTrigger manually triggers a job
func (h *APIHandlers) jobsTrigger(c *gin.Context) {
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only admins can trigger jobs")
		return
	}

	jobName := c.Param("job_name")
	ctx := c.Request.Context()

	// Parse request body for optional parameters
	var req struct {
		Parameters map[string]string `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is OK - no parameters
		req.Parameters = make(map[string]string)
	}

	// Build enqueue options
	var opts []jobs.EnqueueOption
	if len(req.Parameters) > 0 {
		opts = append(opts, jobs.WithParameters(req.Parameters))
	}

	executionUUID, err := h.services.Jobs.EnqueueJob(ctx, jobName, opts...)
	if err != nil {
		mapError := mapJobsError(err)
		if mapError.code != "" {
			respondError(c, http.StatusBadRequest, mapError.code, mapError.message)
			return
		}
		h.getLogger(c).Error("Failed to enqueue job", "err", err, "job", jobName)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"executionUuid": executionUUID,
		"status":        "enqueued",
	})
}

// jobsListRecentExecutions returns recent executions across all jobs
func (h *APIHandlers) jobsListRecentExecutions(c *gin.Context) {
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only admins can view executions")
		return
	}

	ctx := c.Request.Context()

	// Parse limit from query params
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 500 {
		limit = 100
	}

	executions, err := h.services.Jobs.ListRecentExecutions(ctx, limit)
	if err != nil {
		h.getLogger(c).Error("Failed to list recent executions", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, executions)
}

// jobsGetExecution returns details for a specific execution
func (h *APIHandlers) jobsGetExecution(c *gin.Context) {
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only admins can view executions")
		return
	}

	executionUUIDStr := c.Param("execution_uuid")
	executionUUID, err := uuid.Parse(executionUUIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid execution UUID")
		return
	}

	ctx := c.Request.Context()

	execution, err := h.services.Jobs.GetExecution(ctx, executionUUID)
	if err != nil {
		mapError := mapJobsError(err)
		if mapError.code != "" {
			respondError(c, http.StatusNotFound, mapError.code, mapError.message)
			return
		}
		h.getLogger(c).Error("Failed to get execution", "err", err, "execution_uuid", executionUUID)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, execution)
}

// jobsUpdateSettings updates job settings (currently only enabled field)
func (h *APIHandlers) jobsUpdateSettings(c *gin.Context) {
	sessionData := getSessionData(c)
	if !sessionData.Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only admins can update jobs")
		return
	}

	jobName := c.Param("job_name")
	ctx := c.Request.Context()

	var req struct {
		Enabled *bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid request body")
		return
	}

	// Validate that at least one field is provided
	if req.Enabled == nil {
		respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "No fields to update")
		return
	}

	// Update the job
	result := h.services.Jobs.DB().WithContext(ctx).
		Table("jobs").
		Where("name = ?", jobName).
		Update("enabled", *req.Enabled)

	if result.Error != nil {
		h.getLogger(c).Error("Failed to update job", "err", result.Error, "job", jobName)
		c.Status(http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		respondError(c, http.StatusNotFound, ErrCodeJobNotFound, "Job not found")
		return
	}

	c.Status(http.StatusNoContent)
}
