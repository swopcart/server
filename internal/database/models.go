package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	UUID     uuid.UUID `gorm:"uniqueIndex"`
	Username string    `gorm:"uniqueIndex"`

	Password string
	TOTP     *string

	Admin bool

	Sessions []Session
}

type Session struct {
	gorm.Model

	UUID             uuid.UUID `gorm:"uniqueIndex"`
	RefreshTokenHash string    `gorm:"index"`

	UserID uint

	IPAddress string
	UserAgent string

	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

type Job struct {
	gorm.Model `json:"-"`

	UUID uuid.UUID `gorm:"uniqueIndex" json:"uuid"`
	Name string    `gorm:"uniqueIndex" json:"name"`

	Description string `json:"description"`

	// Scheduling
	Schedule *string `json:"schedule"` // Cron expression (null = on-demand only)
	Enabled  bool    `gorm:"default:true" json:"enabled"`

	// Priority & Queue
	Priority int    `gorm:"index;default:0" json:"priority"`
	Queue    string `gorm:"index;default:'default'" json:"queue"`

	// Default parameters (JSON serialized map[string]string)
	DefaultParameters *string `gorm:"type:text" json:"defaultParameters"`

	// Tracking
	LastRunAt *time.Time `json:"lastRunAt"`
	NextRunAt *time.Time `json:"nextRunAt"`

	// Relations
	Executions []JobExecution `json:"executions,omitempty"`
}

type JobExecution struct {
	gorm.Model `json:"-"`

	UUID  uuid.UUID `gorm:"uniqueIndex" json:"uuid"`
	JobID uint      `gorm:"index" json:"jobId"`

	// Parameters used for this execution (JSON serialized map[string]string)
	Parameters *string `gorm:"type:text" json:"parameters"`

	// Execution timing
	StartedAt   time.Time  `gorm:"index" json:"startedAt"`
	CompletedAt *time.Time `json:"completedAt"`
	Duration    *int64     `json:"duration"` // Milliseconds

	// Status and results
	Status string  `gorm:"index" json:"status"` // "pending", "running", "completed", "failed"
	Output *string `gorm:"type:text" json:"output"`
	Error  *string `gorm:"type:text" json:"error"`

	// Trigger context
	TriggerType string `json:"triggerType"` // "scheduled", "manual", "api"

	// Progress tracking
	RecordsComplete int     `json:"recordsComplete"`
	RecordsTotal    int     `json:"recordsTotal"`
	CurrentRecord   *string `gorm:"type:text" json:"currentRecord"`
}
