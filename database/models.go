package database

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	UUID     string `gorm:"uniqueIndex"`
	Username string `gorm:"uniqueIndex"`

	Password string
	TOTP     *string

	Admin bool

	Sessions []Session
}

type Session struct {
	gorm.Model

	UUID      string `gorm:"uniqueIndex"`
	UserAgent string

	UserID uint

	LastUsedAt *time.Time
}
