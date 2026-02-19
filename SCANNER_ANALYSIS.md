# Game Library Scanner Analysis

## Overview
The swopcart library scanner is responsible for:
1. Traversing library directories for ROM files
2. Extracting metadata from filenames
3. Creating/updating games and versions in the database
4. Storing metadata files

## File Locations

### Core Scanner Code
- **Main entry point**: `/home/razz/src/swopcart/server/internal/services/library/jobs.go`
  - `scanLibrary()` (lines 69-275): Main scanning logic
  - `libraryScanJob()` (lines 37-66): Job registration

- **Service**: `/home/razz/src/swopcart/server/internal/services/library/service.go`
  - Platform loading and initialization

- **CRUD Operations**: `/home/razz/src/swopcart/server/internal/services/library/crud.go`
  - `GetOrCreateGame()` (lines 331-368): Game creation logic
  - `UpsertGameVersion()` (lines 371-435): Version creation/update logic
  - `UpdateGameMetadata()` (lines 438-467): Game metadata updates

- **Metadata Handling**: `/home/razz/src/swopcart/server/internal/services/library/metadata_io.go`
  - `ExtractMetadataFromFilename()` (lines 169-237): Filename parsing
  - `LoadMetadataFromFile()` (lines 57-72): Load TOML metadata
  - `SaveMetadataToFile()` (lines 75-92): Save TOML metadata

- **Queries**: `/home/razz/src/swopcart/server/internal/services/library/queries.go`
  - `GetGamesByLibrary()`: Retrieve games with pagination

### Database Models
- `/home/razz/src/swopcart/server/internal/database/models.go` (lines 139-182)
  - `Game` struct (lines 140-158)
  - `GameVersion` struct (lines 161-182)

### API Handlers
- `/home/razz/src/swopcart/server/internal/www/api/v0/games.go`
- `/home/razz/src/swopcart/server/internal/www/api/v0/libraries.go`

## Current Scanning Process (jobs.go: scanLibrary)

### Step 1: Directory Traversal (Two-Pass)
```go
// First pass: Count total files for progress tracking
for _, scanPath := range paths {
    filepath.Walk(scanPath, func(filePath string, info os.FileInfo, err error) error {
        if matchesExtension(filePath, extensions) {
            totalRecords++
        }
    })
}

// Second pass: Process each file
for _, scanPath := range paths {
    filepath.Walk(scanPath, func(filePath string, info os.FileInfo, err error) error {
        if !matchesExtension(filePath, extensions) {
            return nil
        }

        // ... process file
    })
}
```

### Step 2: ROM File Detection
- **Location**: `jobs.go`, lines 127-163
- **Method**: File extension matching against platform's allowed extensions
- **Function**: `matchesExtension(filePath, extensions)` (lines 278-290)
  - Compares file extension (case-insensitive) against platform's extension list
  - Stored in database.Platform.Extensions as JSON array
  - Example: `[".sfc", ".smc", ".rom", ".zip"]` for SNES

### Step 3: Metadata Extraction from Filename
- **Location**: `jobs.go`, line 173
- **Function**: `svc.ExtractMetadataFromFilename(filename, platform.Name)`
- **File**: `metadata_io.go`, lines 169-237

**Patterns Extracted:**
1. **vgdb ID**: `[vgdb=XXXXX]` → stored in ExternalIDs
2. **Region tags**: `.pal`, `.ntsc` → regions
3. **Country codes**: `[USA]`, `[EUR]`, `[JPN]`, etc. → regions
4. **Version/tags**: `(v1.0)`, `(Rev A)`, `(Demo)`, `(Beta)` → tags
5. **Title**: Everything else after removing patterns

**Example:**
- Input: `Super Mario World (PAL) [vgdb=12345].sfc`
- Output:
  ```go
  GameMetadata{
    Title: "Super Mario World",
    Platform: "SNES",
    Regions: []string{"PAL"},
    ExternalIDs: map[string]string{"vgdb": "12345"},
    Tags: []string{},
  }
  ```

### Step 4: Game Creation/Lookup
- **Location**: `jobs.go`, lines 181-185
- **Function**: `GetOrCreateGame(ctx, libraryID, dirPath, metadata.Title)`
- **File**: `crud.go`, lines 331-368

**Current Logic:**
```go
// Try to find existing game by library ID and title
result := svc.db.WithContext(ctx).
    Where("library_id = ? AND title = ?", libraryID, title).
    First(&game)

if result.Error == nil {
    return &game  // Game already exists
}
if result.Error != gorm.ErrRecordNotFound {
    return nil, result.Error
}

// Create new game
game = database.Game{
    ID: uuid.New(),
    LibraryID: libraryID,
    PlatformID: lib.PlatformID,
    Title: title,
}
svc.db.WithContext(ctx).Create(&game)
return &game
```

**KEY ISSUE - BUG ROOT CAUSE:**
- Games are matched ONLY by `(libraryID + title)`
- If two ROMs have different titles extracted from filenames:
  - "Super Mario World (PAL)" → title extracted as "Super Mario World (PAL)"
  - "Super Mario World (NTSC-U)" → title extracted as "Super Mario World (NTSC-U)"
- Each gets a separate Game record, even though they're the same game
- The region info is extracted but NOT used for game matching/grouping

### Step 5: Hash Calculation
- **Location**: `jobs.go`, lines 187-208
- **File**: `hashing.go`, lines 22-52
- **Function**: `CalculateFileHashes(filePath)`
- **Algorithms**: MD5, SHA1, SHA256, Blake3 (reserved)
- **Strategy**: Streaming calculation to avoid loading entire file in memory

**Hash Caching Logic:**
```go
var existingVersion database.GameVersion
if err := svc.db.WithContext(ctx).Where("file_path = ?", filePath).First(&existingVersion).Error; err == gorm.ErrRecordNotFound {
    // New file, calculate hashes
    hashes, err = svc.CalculateFileHashes(filePath)
} else {
    // File already exists, use existing hashes (don't recalculate)
    hashes = &Hashes{
        MD5: existingVersion.MD5,
        SHA1: existingVersion.SHA1,
        SHA256: existingVersion.SHA256,
        Blake3: existingVersion.Blake3,
    }
}
```

### Step 6: Version Creation/Update
- **Location**: `jobs.go`, lines 210-214
- **Function**: `UpsertGameVersion(ctx, gameID, filePath, dirName, fileSize, hashes)`
- **File**: `crud.go`, lines 371-435

**Logic:**
- Creates or updates GameVersion by filePath uniqueness
- VersionName is set to `dirName` (parent directory name)
- **BUG**: Currently uses directory name, not extracted region/version info
  - Should be using extracted metadata regions (PAL, NTSC, etc.)

### Step 7: Metadata File Handling
- **Location**: `jobs.go`, lines 216-222
- **Metadata Path**: `filePath + ".meta"` (e.g., `game.sfc.meta`)

**Process:**
1. Check if metadata file exists
2. If NOT: Create one with extracted metadata
3. If EXISTS: Leave it alone (user may have edited it)

**File Format**: TOML (kebab-case keys)
```toml
title = "Super Mario World"
platform = "SNES"
developer = "Nintendo"
publisher = "Nintendo"
regions = ["PAL"]

[external-ids]
vgdb = "12345"

[[versions]]
name = "PAL"
filename = "Super Mario World (PAL).sfc"
```

**Important**: The `.meta` file is created PER ROM FILE, not per game

### Step 8: Game Soft-Deletion
- **Location**: `jobs.go`, lines 247-271
- **Logic**: Games are soft-deleted if no versions exist on disk

## Data Model Structure

### Game Record
```
Game
├── ID (UUID)
├── LibraryID (FK to Library)
├── PlatformID (FK to Platform)
├── Title (string) ← KEY FOR MATCHING
├── Developer (optional)
├── Publisher (optional)
├── ReleasedDate (optional)
├── Description (optional)
└── Versions (relationship to GameVersion)
```

### GameVersion Record
```
GameVersion
├── ID (UUID)
├── GameID (FK to Game) ← Defines which game this is a variant of
├── VersionName (string) ← Currently set to directory name, should be region
├── FilePath (string) ← Unique key, points to actual ROM file
├── FileSize (int64)
├── MD5/SHA1/SHA256/Blake3 (hashes)
├── MetadataPath (string) ← Path to .meta TOML file
└── MetadataJSON (string) ← JSON-serialized metadata
```

## Metadata Storage

### Two Metadata Sources

**1. Metadata Files (.meta) on Disk**
- Path: `{filePath}.meta` (e.g., `game.sfc.meta`)
- Format: TOML
- Keys: kebab-case (e.g., `release-date`, `external-ids`)
- Created: During scan if doesn't exist
- Scope: Per-ROM-file, not per-game

**2. Database MetadataJSON**
- Stored in: `GameVersion.MetadataJSON` (text field)
- Format: JSON (camelCase for API consistency)
- Purpose: Store version-specific metadata in database
- Currently: Not being populated by scanner

### Conversion Functions
- `ExtractMetadataFromFilename()` → returns GameMetadata struct
- `SaveMetadataToFile()` → writes GameMetadata to TOML
- `LoadMetadataFromFile()` → reads GameMetadata from TOML
- `MetadataToJSON()` → converts to camelCase for API
- `JSONToMetadata()` → converts from camelCase back to kebab-case
- `MetadataToString()` → serializes to JSON string for DB storage

## Platform Extensions Configuration

### Platform Model
```
Platform
├── ID (uint)
├── Name (string) ← e.g., "SNES"
├── Description (string)
└── Extensions (string) ← JSON array: [".sfc", ".smc", ".rom", ".zip"]
```

### Default Platforms Seeded
- SNES: `.sfc`, `.smc`, `.rom`, `.zip`
- NES: `.nes`, `.rom`, `.zip`
- N64: `.z64`, `.n64`, `.rom`, `.zip`
- Atari 2600: `.a26`, `.bin`, `.rom`, `.zip`
- PlayStation 1/2: `.iso`, `.cue`, `.bin`, `.zip`
- Dreamcast: `.iso`, `.cdi`, `.gdi`, `.zip`
- DOS: `.exe`, `.com`, `.bat`, `.zip`
- Windows: `.exe`, `.iso`, `.zip`

## Directory Structure Assumptions

### Expected Layout
```
library_path/
├── game-1.sfc              # ROM file
├── game-1.sfc.meta         # Auto-created metadata
├── game-2.sfc              # Another game
├── game-2.sfc.meta
├── subdirectory/
│   ├── game-3.sfc
│   └── game-3.sfc.meta
└── ...
```

**Key Points:**
- No assumption of directory-per-game structure
- Supports flat directory with multiple ROMs
- Supports nested subdirectories (recursively walked)
- Metadata files are created adjacent to ROM files

## Current Bug: Multiple Games for Same Title

### Symptom
Two files "Super Mario World (PAL).sfc" and "Super Mario World (NTSC-U).sfc" create TWO separate games instead of ONE game with 2 versions.

### Root Cause
In `crud.go:331-368` (`GetOrCreateGame`):
```go
// Matches ONLY on title
Where("library_id = ? AND title = ?", libraryID, title)
```

The title includes region info from filename:
- File 1: "Super Mario World (PAL).sfc" → title = "Super Mario World (PAL)"
- File 2: "Super Mario World (NTSC-U).sfc" → title = "Super Mario World (NTSC-U)"

Different titles = different Game records

### Why It's Wrong
1. Extracted metadata ALREADY contains regions: `metadata.Regions = ["PAL"]` or `["NTSC"]`
2. But the region is NOT removed from title
3. Game grouping ignores the extracted regions
4. Game grouping ignores hash-based matching (could group by content)

### Expected Behavior
1. Extract title WITHOUT region: "Super Mario World"
2. Use cleaned title for game matching
3. Store region in version metadata or Game record
4. Multiple ROMs with same clean title → same Game, different Versions
5. Versions differentiated by VersionName = "PAL", "NTSC-U", etc.

## Key Design Patterns

### Extension Filtering
- Stored as JSON in database.Platform.Extensions
- Parsed at runtime into []string
- Case-insensitive matching

### Progress Tracking
- Two-pass directory walk (count, then process)
- `progress.UpdateProgress(total, completed, currentFile)`
- Stored in JobExecution.RecordsComplete/RecordsTotal

### Hashing Strategy
- Stream-based to avoid memory issues on low-RAM devices
- Single-pass multi-writer (MD5, SHA1, SHA256 calculated simultaneously)
- Hashes only calculated on first scan
- Cached on subsequent scans to avoid recalculation

### Soft Deletion
- Games soft-deleted if all versions removed
- Uses GORM's soft delete (gorm.DeletedAt field)
- Can be undeleted if files reappear

## API Response Format

### GameWithVersions Response
```json
{
  "id": "uuid",
  "title": "Super Mario World",
  "platformId": 2,
  "developer": "Nintendo",
  "publisher": "Nintendo",
  "releasedDate": "1990-11-21",
  "description": "...",
  "versions": [
    {
      "id": "uuid",
      "versionName": "PAL",
      "filePath": "/path/to/game.sfc",
      "fileSize": 1048576,
      "md5": "...",
      "sha1": "...",
      "sha256": "...",
      "metadataJson": { ... }
    },
    {
      "id": "uuid",
      "versionName": "NTSC-U",
      "filePath": "/path/to/game-us.sfc",
      "fileSize": 1048576,
      "md5": "...",
      "metadataJson": { ... }
    }
  ],
  "metadataJson": {
    "developer": "Nintendo",
    "publisher": "Nintendo"
  }
}
```

## Summary of What Needs Fixing

1. **Title Extraction**: Remove region/variant info from title before game matching
2. **Game Matching**: Use clean title (without regions) to group ROMs
3. **Version Naming**: Use extracted region/variant as VersionName
4. **Metadata Storage**: Consider using GameVersion.MetadataJSON to store per-version metadata
5. **Hash-Based Matching**: Optional: Could also match games by file hash for better deduplication

## Files Summary

| File | Purpose | Key Functions |
|------|---------|---|
| jobs.go | Main scanning orchestration | `scanLibrary()`, `matchesExtension()` |
| service.go | Service initialization | `NewLibraryService()`, `loadPlatforms()` |
| crud.go | Game/Version CRUD | `GetOrCreateGame()`, `UpsertGameVersion()` |
| metadata_io.go | Filename parsing & file I/O | `ExtractMetadataFromFilename()` |
| queries.go | Game retrieval | `GetGamesByLibrary()`, `GetGameByID()` |
| hashing.go | Hash calculation | `CalculateFileHashes()` |
| models.go | Database schemas | `Game`, `GameVersion`, `Platform`, `Library` |
| games.go | API handlers | `ListGamesHandler()`, `GetGameHandler()` |
| libraries.go | Library API handlers | `CreateLibraryHandler()`, `TriggerLibraryScanHandler()` |
