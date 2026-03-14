package library_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pelletier/go-toml/v2"
	"github.com/swopcart/server/internal/services/library"
)

func TestMetadataTOMLMarshal(t *testing.T) {
	original := library.Metadata{
		Title:      "Celeste 64: Fragments of the Mountain",
		Developer:  "Maddy Thorson, Noel Berry, Amora B., Pedro Medeiros, Power Up Audio, Lena Raine, Heidy Motta",
		Publisher:  "EXOK",
		ReleasedAt: library.SimpleDate{Time: time.Date(2024, 1, 30, 0, 0, 0, 0, time.UTC)},
		Versions: map[string]library.Version{
			"win-111": {
				Name:     "Windows v1.1.1",
				Filename: "Celeste64-v1.1.1-Windows-x64.zip",
				Platform: "pc:win32:2025:amd64",
				Labels:   []string{"zipped"},
				RunnerMeta: map[string]string{
					"executable": "Celeste64.exe",
				},
			},
			"linux-111": {
				Name:     "Linux v1.1.1",
				Filename: "Celeste64-v1.1.1-Linux-x64.zip",
				Platform: "pc:linux:2025:amd64",
				Labels:   []string{"zipped"},
				RunnerMeta: map[string]string{
					"executable": "Celeste64",
				},
			},
		},
	}

	// Marshal to TOML
	tomlBytes, err := toml.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal to TOML: %v", err)
	}

	// Unmarshal back to verify round-trip
	var roundTripped library.Metadata
	err = toml.Unmarshal(tomlBytes, &roundTripped)
	if err != nil {
		t.Fatalf("Failed to unmarshal TOML: %v", err)
	}

	if diff := cmp.Diff(original, roundTripped); diff != "" {
		t.Errorf("TOML round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestMetadataJSONMarshal(t *testing.T) {
	original := library.Metadata{
		Title:      "Celeste 64: Fragments of the Mountain",
		Developer:  "Maddy Thorson, Noel Berry, Amora B., Pedro Medeiros, Power Up Audio, Lena Raine, Heidy Motta",
		Publisher:  "EXOK",
		ReleasedAt: library.SimpleDate{Time: time.Date(2024, 1, 30, 0, 0, 0, 0, time.UTC)},
		Versions: map[string]library.Version{
			"win-111": {
				Name:     "Windows v1.1.1",
				Filename: "Celeste64-v1.1.1-Windows-x64.zip",
				Platform: "pc:win32:2025:amd64",
				Labels:   []string{"zipped"},
				RunnerMeta: map[string]string{
					"executable": "Celeste64.exe",
				},
			},
			"linux-111": {
				Name:     "Linux v1.1.1",
				Filename: "Celeste64-v1.1.1-Linux-x64.zip",
				Platform: "pc:linux:2025:amd64",
				Labels:   []string{"zipped"},
				RunnerMeta: map[string]string{
					"executable": "Celeste64",
				},
			},
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	// Unmarshal back to verify round-trip
	var roundTripped library.Metadata
	err = json.Unmarshal(jsonBytes, &roundTripped)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if diff := cmp.Diff(original, roundTripped); diff != "" {
		t.Errorf("JSON round-trip mismatch (-want +got):\n%s", diff)
	}
}
