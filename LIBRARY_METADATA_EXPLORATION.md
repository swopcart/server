# Swopcart Library Structure & Metadata Exploration

## Executive Summary

The swopcart library system is designed to:
1. **Manage multi-version games** across diverse platforms (retro to modern)
2. **Store metadata** in both TOML files (on disk) and database records
3. **Track game versions** as separate database records with file hashes
4. **Support flexible directory structures** (flat files or per-game directories)

There are **two separate metadata systems** in the codebase:
- **Original metadata.go** (legacy, unused): Versioned metadata with keyed versions (see metadata_test.go)
- **Active metadata_io.go**: GameMetadata used for scanning (used by jobs.go)

---

## 1. Current Library Directory Structure Expectations

### Pattern 1: Flat File Structure
```
/mnt/games/nes/
├── zelda.nes
├── zelda.nes.meta           # TOML metadata file (kebab-case)
├── mario.nes
├── mario.nes.meta
└── donkey_kong.nes
```

**Key Points:**
- One metadata file per ROM file
- Naming: `{game-filename}.meta`
- Location: Same directory as the ROM

### Pattern 2: Per-Game Directory Structure
```
/mnt/games/nes/
├── zelda/
│   ├── zelda_1.0.nes
│   ├── zelda_pal.nes
│   └── meta.toml            # Shared metadata for directory
├── mario/
│   ├── mario.nes
│   └── meta.toml
└── donkey_kong.nes
```

**Key Points:**
- One directory per game (optional)
- Multiple versions in same directory
- Shared `meta.toml` file
- Flat files at library root also supported

### How Scanner Handles Structure (jobs.go lines 127-245)

```
1. filepath.Walk() traverses all paths
2. For each file matching platform extensions:
   - dirName = filepath.Dir(filePath)   # e.g., "/games/nes" or "/games/nes/zelda"
   - filename = filepath.Base(filePath) # e.g., "zelda.nes"
3. Creates ONE game per directory/title combo
4. Creates ONE GameVersion per actual file
5. Metadata saved to {filePath}.meta (sibling file)
```

**Current Limitation:** Scanner creates metadata per file (`filePath + ".meta"`), NOT per directory
- This doesn't match the "per-game directory" pattern shown in GAME_LIBRARY_PLAN.md

---

## 2. Metadata File Format (TOML) - Complete Schema

### File Location
- **Flat files**: `{romfile}.meta` (e.g., `zelda.nes.meta`)
- **Directory-based**: `meta.toml` inside game directory
- **Format**: TOML with kebab-case keys (for file on disk)

### TOML Schema (On Disk - kebab-case)

```toml
title = "The Legend of Zelda"
platform = "NES"
developer = "Nintendo"
publisher = "Nintendo"
release-date = "1986-07-27"
description = "Epic adventure game"

[external-ids]
vgdb = "12345"
igdb = "67890"

regions = ["NTSC", "PAL"]
tags = ["action", "adventure", "classic"]

[[versions]]
name = "1.0"
filename = "zelda_1.0.nes"
regions = ["NTSC"]

[[versions]]
name = "PAL"
filename = "zelda_pal.nes"
regions = ["PAL"]
```

### Go Struct (metadata_io.go lines 14-33)

```go
type GameMetadata struct {
    Title       string            `toml:"title"`
    Platform    string            `toml:"platform,omitempty"`
    Developer   string            `toml:"developer,omitempty"`
    Publisher   string            `toml:"publisher,omitempty"`
    ReleaseDate string            `toml:"release-date,omitempty"` // YYYY-MM-DD
    Description string            `toml:"description,omitempty"`
    ExternalIDs map[string]string `toml:"external-ids,omitempty"`
    Regions     []string          `toml:"regions,omitempty"`
    Tags        []string          `toml:"tags,omitempty"`
    Versions    []VersionMetadata `toml:"versions,omitempty"`
}

type VersionMetadata struct {
    Name     string   `toml:"name"`
    Filename string   `toml:"filename"`
    Regions  []string `toml:"regions,omitempty"`
}
```

### API Response (JSON - camelCase)

```json
{
  "title": "The Legend of Zelda",
  "platform": "NES",
  "developer": "Nintendo",
  "publisher": "Nintendo",
  "releaseDate": "1986-07-27",
  "description": "Epic adventure game",
  "externalIds": {
    "vgdb": "12345",
    "igdb": "67890"
  },
  "regions": ["NTSC", "PAL"],
  "tags": ["action", "adventure", "classic"],
  "versions": [
    {
      "name": "1.0",
      "filename": "zelda_1.0.nes",
      "regions": ["NTSC"]
    },
    {
      "name": "PAL",
      "filename": "zelda_pal.nes",
      "regions": ["PAL"]
    }
  ]
}
```

### Go Struct (API Response - metadata_io.go lines 35-54)

```go
type GameMetadataJSON struct {
    Title       string                `json:"title"`
    Platform    string                `json:"platform,omitempty"`
    Developer   string                `json:"developer,omitempty"`
    Publisher   string                `json:"publisher,omitempty"`
    ReleaseDate string                `json:"releaseDate,omitempty"`
    Description string                `json:"description,omitempty"`
    ExternalIDs map[string]string     `json:"externalIds,omitempty"`
    Regions     []string              `json:"regions,omitempty"`
    Tags        []string              `json:"tags,omitempty"`
    Versions    []VersionMetadataJSON `json:"versions,omitempty"`
}

type VersionMetadataJSON struct {
    Name     string   `json:"name"`
    Filename string   `json:"filename"`
    Regions  []string `json:"regions,omitempty"`
}
```

---

## 3. Database Models for Game & GameVersion

### Game Model (models.go lines 140-158)

```go
type Game struct {
    ID        uuid.UUID `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
    DeletedAt gorm.DeletedAt  // Soft-delete support

    LibraryID  uuid.UUID `gorm:"index" json:"libraryId"`
    Library    Library   `gorm:"foreignKey:LibraryID" json:"-"`
    PlatformID uint      `json:"platformId"`
    Platform   Platform  `gorm:"foreignKey:PlatformID" json:"-"`

    Title        string     `gorm:"index" json:"title"`       // Clean title from filename
    Developer    *string    `json:"developer"`                // Optional
    Publisher    *string    `json:"publisher"`                // Optional
    ReleasedDate *time.Time `json:"releasedDate"`             // Optional
    Description  *string    `json:"description"`              // Optional

    Versions []GameVersion `gorm:"foreignKey:GameID" json:"versions,omitempty"`
}
```

**Key Points:**
- One Game per directory/title combination
- Title is indexed for fast lookup
- Platform links to Platform table for extensions
- Soft-delete via `gorm.DeletedAt` (games not physically deleted)

### GameVersion Model (models.go lines 161-182)

```go
type GameVersion struct {
    ID        uuid.UUID `gorm:"primaryKey" json:"id"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`

    GameID      uuid.UUID `gorm:"index" json:"gameId"`
    Game        Game      `gorm:"foreignKey:GameID" json:"-"`
    VersionName string    `json:"versionName"`                 // "1.0", "PAL", "NTSC", etc.
    FilePath    string    `gorm:"uniqueIndex" json:"filePath"` // Full absolute path to file
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
```

**Key Points:**
- One record per actual ROM file
- FilePath is unique index (each file has one version record)
- VersionName extracted from directory name or filename pattern
- Hashes calculated only on first scan, not recalculated on updates
- MetadataJSON stores serialized GameMetadataJSON for quick API responses

### Relationship Diagram

```
Library (1 to many) Game
  |                    |
  +-- Paths: []string  +-- Versions: []GameVersion (1 to many)
  +-- PlatformID           |
       |                   +-- VersionName: string
       |                   +-- FilePath: string (unique)
       v                   +-- MD5, SHA1, SHA256, Blake3: string
    Platform
      |
      +-- Extensions: []string
      +-- Name: string
```

---

## 4. Current metadata.go Implementation (LEGACY - Unused)

**Location**: `internal/services/library/metadata.go` (lines 63-88)

**IMPORTANT:** This is NOT used by the scanner. It's a different structure tested in metadata_test.go.

```go
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
    Name     string `json:"name"     toml:"name"`
    Filename string `json:"filename" toml:"filename"`
    Platform string `json:"platform,omitempty" toml:"platform,omitempty"`

    Labels     []string          `json:"labels"     toml:"labels"`
    RunnerMeta map[string]string `json:"runnerMeta" toml:"runner-meta"`

    MD5    string `json:"md5,omitempty"    toml:"md5,omitempty"`
    SHA1   string `json:"sha1,omitempty"   toml:"sha1,omitempty"`
    SHA256 string `json:"sha256,omitempty" toml:"sha256,omitempty"`
    Blake3 string `json:"blake3,omitempty" toml:"blake3,omitempty"`
}
```

**Key Differences from GameMetadata:**
- Has a UUID field (instance identifier)
- Versions keyed by string (map[string]Version) instead of array
- Platform per version (per-version runner support)
- Labels and RunnerMeta (for emulator/runner configuration)
- Hashes stored in version record itself

**Used For:**
- Testing TOML marshaling (metadata_test.go)
- Possibly future versioned metadata system

---

## 5. Metadata I/O Operations (metadata_io.go)

### Loading from File

```go
func (svc *LibraryService) LoadMetadataFromFile(filePath string) (*GameMetadata, error) {
    // Returns nil (not error) if file doesn't exist
    // Parses TOML and returns GameMetadata struct
}
```

### Saving to File

```go
func (svc *LibraryService) SaveMetadataToFile(filePath string, metadata *GameMetadata) error {
    // Marshals to TOML
    // Creates directory if needed
    // Writes with 0644 permissions
}
```

### Format Conversions

```go
// TOML → JSON (with case conversion)
func (svc *LibraryService) MetadataToJSON(metadata *GameMetadata) *GameMetadataJSON

// JSON → TOML (with case conversion)
func (svc *LibraryService) JSONToMetadata(jsonMeta *GameMetadataJSON) *GameMetadata

// Serialize to JSON string for database storage
func (svc *LibraryService) MetadataToString(metadata *GameMetadata) (string, error)

// Deserialize from JSON string
func (svc *LibraryService) StringToMetadata(metadataStr string) (*GameMetadata, error)
```

---

## 6. Filename Parsing Rules (metadata_io.go lines 169-260)

The scanner extracts metadata from filenames using these patterns:

### Extraction Patterns

| Pattern | Example | Result | Field |
|---------|---------|--------|-------|
| `[vgdb=XXXXX]` | `Zelda [vgdb=12345].nes` | `"12345"` | `ExternalIDs["vgdb"]` |
| `.pal` | `Mario.pal.sfc` | `"PAL"` | `Regions[]` |
| `.ntsc` | `Mario.ntsc.sfc` | `"NTSC"` | `Regions[]` |
| `.ntsc-u` | `Mario.ntsc-u.sfc` | `["NTSC", "USA"]` | `Regions[]` |
| `.ntsc-j` | `Mario.ntsc-j.sfc` | `["NTSC", "JPN"]` | `Regions[]` |
| `[USA]`, `[EUR]`, `[JPN]` | `Game [USA].nes` | `"USA"` | `Regions[]` |
| `(v1.0)`, `(Rev A)` | `Game (v1.0).nes` | Removed from title | (Discarded currently) |
| `(Demo)`, `(Beta)` | `Game (Demo).nes` | `"demo"` | `Tags[]` |
| Everything else | `Super Mario World` | Used as-is | `Title` |

### Example Parsing

**Input filename:** `Super Mario World (PAL) [vgdb=12345].sfc`

**Extraction steps:**
1. Remove extension: `Super Mario World (PAL) [vgdb=12345]`
2. Extract vgdb: `ExternalIDs["vgdb"] = "12345"`, title becomes `Super Mario World (PAL) `
3. Extract region: `Regions = ["PAL"]`, title becomes `Super Mario World `
4. Extract version (discarded): `(PAL)` removed
5. Clean spaces: `title = "Super Mario World"`

**Output:**
```go
GameMetadata{
    Title: "Super Mario World",
    Platform: "SNES",
    Regions: []string{"PAL"},
    ExternalIDs: map[string]string{"vgdb": "12345"},
    Tags: []string{},
}
```

---

## 7. Scanner Flow (jobs.go: scanLibrary)

### Complete Scan Process

```
1. PREPARE
   ├── Check if scan already in progress (prevent race)
   ├── Mark library as "scanning"
   └── Get library paths, platform, extensions

2. COUNT FILES (First Pass)
   ├── Walk all paths
   ├── Count files matching extensions
   └── Update progress.UpdateProgress(total, 0, "")

3. PROCESS FILES (Second Pass)
   ├── Walk all paths again
   ├── For each matching file:
   │   ├── Track in foundFiles map
   │   ├── Get directory name → game title candidate
   │   ├── ExtractMetadataFromFilename(filename, platform)
   │   ├── GetOrCreateGame(libraryID, dirPath, title)
   │   ├── CalculateFileHashes (first time only)
   │   ├── UpsertGameVersion(gameID, filePath, versionName, hashes)
   │   ├── SaveMetadataToFile(filePath + ".meta", metadata)
   │   └── UpdateGameMetadata (if extracted developer/publisher)

4. CLEANUP
   ├── Soft-delete games no longer on disk
   └── Mark library as "idle"
```

### Hash Calculation Strategy (hashing.go)

```go
func (svc *LibraryService) CalculateFileHashes(filePath string) (*Hashes, error) {
    // Stream reading to avoid loading entire file in memory
    // Calculate MD5, SHA1, SHA256 simultaneously
    // Returns struct with all four hashes
}
```

**Hash Caching (lines 188-208):**
```go
// Check if version already exists
if err := svc.db.WithContext(ctx).Where("file_path = ?", filePath).First(&existingVersion).Error; err == gorm.ErrRecordNotFound {
    // NEW FILE: Calculate hashes
    hashes, err = svc.CalculateFileHashes(filePath)
} else if err != nil {
    // ERROR: Use empty hashes
    hashes = &Hashes{}
} else {
    // EXISTS: Reuse existing hashes (don't recalculate)
    hashes = &Hashes{
        MD5:    existingVersion.MD5,
        SHA1:   existingVersion.SHA1,
        SHA256: existingVersion.SHA256,
        Blake3: existingVersion.Blake3,
    }
}
```

**Key Point:** Hashes only calculated on first scan; subsequent scans reuse existing values.

---

## 8. Game Matching Logic (CURRENT - Simple, Non-Versioned)

### Current: Title-Based Matching (crud.go lines 331-368)

```go
func (svc *LibraryService) GetOrCreateGame(ctx context.Context, libraryID uuid.UUID, dirPath string, title string) (*database.Game, error) {
    // Match game by: libraryID + title (only!)
    result := svc.db.WithContext(ctx).
        Where("library_id = ? AND title = ?", libraryID, title).
        First(&game)

    if result.Error == nil {
        return &game  // Found existing game
    }

    if result.Error != gorm.ErrRecordNotFound {
        return nil, result.Error
    }

    // Create new game if not found
    game = database.Game{
        ID:         uuid.New(),
        LibraryID:  libraryID,
        PlatformID: lib.PlatformID,
        Title:      title,  // Extracted from filename
    }
    // ...create in DB
    return &game
}
```

### Problem: Duplicate Games

If two ROM files have **different titles** extracted from filenames, they create **separate games**:

```
Files:
  ├── Super Mario World.pal.sfc      → Title: "Super Mario World"    → Game A
  └── Super Mario World.ntsc-u.sfc   → Title: "Super Mario World"    → Game A (same)

But if filenames include region:
  ├── Super Mario World (PAL).sfc    → Title: "Super Mario World (PAL)" → Game A
  └── Super Mario World (NTSC).sfc   → Title: "Super Mario World (NTSC)" → Game B (different!)
```

**Metadata Usage:** Extracted metadata (developer, publisher, regions) is captured but NOT used for game matching/grouping.

---

## 9. Platforms & Extensions (service.go)

### Platform Table

```go
type Platform struct {
    ID          uint      `gorm:"primaryKey"`
    CreatedAt   time.Time
    UpdatedAt   time.Time

    Name        string   `gorm:"uniqueIndex"`  // "NES", "SNES", etc.
    Description string
    Extensions  *string  `gorm:"type:text"`    // JSON array: [".nes", ".rom", ".zip"]
}
```

### Default Platforms (service.go lines 111-131)

```go
{
    {"NES", "Nintendo Entertainment System", []string{".nes", ".rom", ".zip"}},
    {"SNES", "Super Nintendo Entertainment System", []string{".sfc", ".smc", ".rom", ".zip"}},
    {"N64", "Nintendo 64", []string{".z64", ".n64", ".rom", ".zip"}},
    {"Game Boy", "Nintendo Game Boy", []string{".gb", ".rom", ".zip"}},
    // ... more platforms ...
    {"DOS", "MS-DOS", []string{".exe", ".com", ".bat", ".zip"}},
}
```

### File Matching (jobs.go lines 278-290)

```go
func matchesExtension(filePath string, extensions []string) bool {
    ext := strings.ToLower(filepath.Ext(filePath))  // Case-insensitive
    for _, allowedExt := range extensions {
        if strings.EqualFold(ext, allowedExt) {
            return true
        }
    }
    return false
}
```

---

## 10. Version Tracking

### How Versions Are Identified

**Version name extracted from:**
1. **Directory name** (if game is in subdirectory)
   - `games/nes/zelda/zelda_1.0.nes` → VersionName: `"zelda"`
2. **Filename pattern** (not currently used)
   - `(v1.0)`, `(PAL)`, etc. removed but not stored

**Current Implementation (jobs.go line 211):**
```go
_, err = svc.UpsertGameVersion(ctx, game.ID, filePath, dirName, info.Size(), hashes.ToMap())
//                                                         ^^^^^^^
//                                                  Uses directory name as version name
```

### GameVersion Fields

| Field | Source | Purpose |
|-------|--------|---------|
| `ID` | Generated UUID | Primary key |
| `GameID` | Parent Game | Foreign key |
| `FilePath` | Filesystem | Full path; unique index |
| `VersionName` | Directory name | Display name |
| `FileSize` | `os.FileInfo` | Bytes |
| `MD5/SHA1/SHA256/Blake3` | CalculateFileHashes | File verification |
| `MetadataPath` | `filePath + ".meta"` | Path to metadata file |
| `MetadataJSON` | GameMetadata serialized | Cached API response |

---

## 11. Real-World Example

### Library Structure

```
/mnt/games/snes/
├── Super Mario World.sfc
├── Super Mario World.sfc.meta      # Flat file metadata
├── Zelda/
│   ├── zelda_1.0.sfc
│   ├── zelda_1.0.sfc.meta
│   ├── zelda_pal.sfc
│   └── zelda_pal.sfc.meta
└── Donkey Kong Country [USA].sfc
```

### Scanner Processing

**File 1: Super Mario World.sfc**
```
1. Filename parsing: "Super Mario World"
2. Metadata extracted: title="Super Mario World", regions=[], tags=[]
3. Game created: Game{Title: "Super Mario World"}
4. Version created: GameVersion{
     VersionName: "snes",  // dirname
     FilePath: "/mnt/games/snes/Super Mario World.sfc",
     MetadataPath: "/mnt/games/snes/Super Mario World.sfc.meta",
   }
5. Metadata saved: /mnt/games/snes/Super Mario World.sfc.meta
```

**File 2: Zelda/zelda_1.0.sfc**
```
1. Filename parsing: "zelda_1"  (note: underscore)
2. Metadata extracted: title="zelda_1", regions=[], tags=[]
3. Game created: Game{Title: "zelda_1"}  (separate from zelda_pal!)
4. Version created: GameVersion{
     VersionName: "Zelda",  // dirname
     FilePath: "/mnt/games/snes/Zelda/zelda_1.0.sfc",
     MetadataPath: "/mnt/games/snes/Zelda/zelda_1.0.sfc.meta",
   }
```

**File 3: Zelda/zelda_pal.sfc**
```
1. Filename parsing: "zelda" (after removing .pal)
2. Regions: ["PAL"]
3. Game created: Game{Title: "zelda"}  (DIFFERENT from zelda_1!)
4. Version created: GameVersion{
     VersionName: "Zelda",  // dirname (SAME as zelda_1.0)
     FilePath: "/mnt/games/snes/Zelda/zelda_pal.sfc",
   }
```

**Result:** Two separate games even though both are versions of Zelda!

---

## Summary Table

| Aspect | Current Implementation | Location |
|--------|----------------------|----------|
| **Directory Traversal** | Two-pass filepath.Walk | jobs.go:126-245 |
| **Metadata Load** | TOML from `{filePath}.meta` | metadata_io.go:57-72 |
| **Metadata Save** | TOML to `{filePath}.meta` | metadata_io.go:75-92 |
| **Filename Parsing** | Regex-based pattern extraction | metadata_io.go:169-260 |
| **Game Matching** | libraryID + title (simple) | crud.go:331-368 |
| **Version Creation** | One per file, named by dirname | jobs.go:211 |
| **Hash Calculation** | Streaming, cached | hashing.go:22-52, jobs.go:187-208 |
| **Metadata Format** | TOML on disk, JSON in DB | metadata_io.go:14-54 |
| **Case Conversion** | TOML kebab-case ↔ JSON camelCase | metadata_io.go:95-142 |
