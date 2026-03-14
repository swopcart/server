package library

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SimpleDate represents a calendar date (YYYY-MM-DD) without time or timezone.
type SimpleDate struct {
	time.Time
}

// UnmarshalText parses a date string in YYYY-MM-DD format.
func (d *SimpleDate) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*d = SimpleDate{}
		return nil
	}

	t, err := time.Parse("2006-01-02", string(text))
	if err != nil {
		return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %w", err)
	}

	*d = SimpleDate{t}
	return nil
}

// MarshalText formats the date as YYYY-MM-DD.
func (d SimpleDate) MarshalText() ([]byte, error) {
	if d.IsZero() {
		return nil, nil
	}
	return []byte(d.Format("2006-01-02")), nil
}

// UnmarshalJSON parses a JSON date string in YYYY-MM-DD format.
func (d *SimpleDate) UnmarshalJSON(data []byte) error {
	// Handle null
	if string(data) == "null" {
		*d = SimpleDate{}
		return nil
	}

	// Remove quotes from JSON string
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}

	return d.UnmarshalText(data)
}

// MarshalJSON formats the date as a JSON string in YYYY-MM-DD format.
func (d SimpleDate) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + d.Format("2006-01-02") + `"`), nil
}

type Metadata struct {
	UUID uuid.UUID `json:"uuid,omitempty" toml:"uuid,omitempty"` // Instance UUID

	Title    string `json:"title"              toml:"title"`
	Platform string `json:"platform,omitempty" toml:"platform,omitempty"`

	Developer  string     `json:"developer"  toml:"developer"`
	Publisher  string     `json:"publisher"  toml:"publisher"`
	ReleasedAt SimpleDate `json:"releasedAt" toml:"released-at"`

	Versions map[string]Version `json:"versions" toml:"versions"`
}

type Version struct {
	Name     string `json:"name"     toml:"name"`     // User-visible name
	Filename string `json:"filename" toml:"filename"` // Version file name
	Platform string `json:"platform,omitempty" toml:"platform,omitempty"`

	Labels     []string          `json:"labels"     toml:"labels"`
	RunnerMeta map[string]string `json:"runnerMeta" toml:"runner-meta"`

	MD5    string `json:"md5,omitempty"    toml:"md5,omitempty"`    // MD5 file hash
	SHA1   string `json:"sha1,omitempty"   toml:"sha1,omitempty"`   // SHA1 file hash
	SHA256 string `json:"sha256,omitempty" toml:"sha256,omitempty"` // SHA256 file hash
	Blake3 string `json:"blake3,omitempty" toml:"blake3,omitempty"` // Blake3 file hash
}
