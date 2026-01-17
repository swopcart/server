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

	UUID      uuid.UUID `gorm:"uniqueIndex"`
	UserAgent string

	UserID uint

	LastUsedAt *time.Time
}
