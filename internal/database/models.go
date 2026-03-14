package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The models
var Models = []any{
	&User{},
	&Session{},
	&Job{},
	&JobExecution{},
	&Platform{},
	&Library{},
	&Game{},
	&GameVersion{},
}

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

// Platform represents a gaming platform (NES, SNES, DOS, etc.)
type Platform struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Name        string  `gorm:"uniqueIndex" json:"name"` // "NES", "SNES", etc.
	Description string  `json:"description"`
	Extensions  *string `gorm:"type:text" json:"extensions"` // JSON array of extensions: [".nes", ".rom"]
}

// Library represents a game library (collection of games from a platform in specific directories)
type Library struct {
	ID        uuid.UUID `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	DeletedAt gorm.DeletedAt

	Name        string   `json:"name"`
	Description string   `json:"description"`
	PlatformID  uint     `json:"platformId"`
	Platform    Platform `gorm:"foreignKey:PlatformID" json:"-"`

	// JSON array of directory paths: ["/mnt/games/nes", "/data/roms"]
	Paths *string `gorm:"type:text" json:"paths"`

	// Scan status tracking
	LastScannedAt    *time.Time `json:"lastScannedAt"`
	ScanStatus       string     `gorm:"default:'idle'" json:"scanStatus"` // "idle", "scanning", "error"
	LastScanError    *string    `json:"lastScanError"`
	CurrentScanJobID *uuid.UUID `json:"currentScanJobId"` // Track active scan to prevent concurrent scans
}

// Game represents a game title (one per directory or flat file)
type Game struct {
	ID        uuid.UUID `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	DeletedAt gorm.DeletedAt

	LibraryID  uuid.UUID `gorm:"index" json:"libraryId"`
	Library    Library   `gorm:"foreignKey:LibraryID" json:"-"`
	PlatformID uint      `json:"platformId"`
	Platform   Platform  `gorm:"foreignKey:PlatformID" json:"-"`

	Title        string     `gorm:"index" json:"title"` // Clean title from filename
	Developer    *string    `json:"developer"`
	Publisher    *string    `json:"publisher"`
	ReleasedDate *time.Time `json:"releasedDate"`
	Description  *string    `json:"description"`

	Versions []GameVersion `gorm:"foreignKey:GameID" json:"versions,omitempty"`
}

// GameVersion represents a specific version/variant of a game
type GameVersion struct {
	ID        uuid.UUID `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	GameID      uuid.UUID `gorm:"index" json:"gameId"`
	Game        Game      `gorm:"foreignKey:GameID" json:"-"`
	VersionName string    `json:"versionName"`                 // "1.0", "PAL", "NTSC", "Demo", etc.
	FilePath    string    `gorm:"uniqueIndex" json:"filePath"` // Full path to file
	FileSize    int64     `json:"fileSize"`

	// Hashes calculated on first scan only
	MD5    string `json:"md5"`    // 32 hex chars
	SHA1   string `json:"sha1"`   // 40 hex chars
	SHA256 string `json:"sha256"` // 64 hex chars
	Blake3 string `json:"blake3"` // 64 hex chars

	MetadataPath string `json:"metadataPath"` // Path to .meta file

	// JSON representation of TOML metadata for API responses
	MetadataJSON *string `gorm:"type:text" json:"metadataJson"`
}
