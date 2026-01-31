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
	gorm.Model

	UUID uuid.UUID `gorm:"uniqueIndex"`
	Name string    `gorm:"uniqueIndex"`

	Description string

	// Scheduling
	Schedule *string // Cron expression (null = on-demand only)
	Enabled  bool    `gorm:"default:true"`

	// Priority & Queue
	Priority int    `gorm:"index;default:0"`
	Queue    string `gorm:"index;default:'default'"`

	// Default parameters (JSON serialized map[string]string)
	DefaultParameters *string `gorm:"type:text"`

	// Tracking
	LastRunAt *time.Time
	NextRunAt *time.Time

	// Relations
	Executions []JobExecution
}

type JobExecution struct {
	gorm.Model

	UUID  uuid.UUID `gorm:"uniqueIndex"`
	JobID uint      `gorm:"index"`

	// Parameters used for this execution (JSON serialized map[string]string)
	Parameters *string `gorm:"type:text"`

	// Execution timing
	StartedAt   time.Time `gorm:"index"`
	CompletedAt *time.Time
	Duration    *int64 // Milliseconds

	// Status and results
	Status string  `gorm:"index"` // "pending", "running", "completed", "failed"
	Output *string `gorm:"type:text"`
	Error  *string `gorm:"type:text"`

	// Trigger context
	TriggerType string // "scheduled", "manual", "api"

	// Progress tracking
	RecordsComplete int
	RecordsTotal    int
	CurrentRecord   *string `gorm:"type:text"`
}
