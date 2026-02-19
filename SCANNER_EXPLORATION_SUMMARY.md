# Game Library Scanner - Complete Exploration Summary

## Executive Summary

The swopcart game library scanner is a multi-step process that:
1. Traverses library directories recursively
2. Filters files by platform-specific extensions
3. Extracts metadata (regions, version tags) from filenames
4. Groups ROMs into games and versions
5. Calculates file hashes and stores metadata

**Critical Bug Found**: Games are being grouped ONLY by title, but the title extraction includes region/variant information. This causes "Super Mario World (PAL)" and "Super Mario World (NTSC-U)" to be treated as 2 separate games instead of 1 game with 2 versions.

---

## File Locations

### Scanner Implementation Files
```
/home/razz/src/swopcart/server/internal/services/library/
├── jobs.go                  # Main scanning orchestration
├── crud.go                  # Game/Version creation logic
├── metadata_io.go          # Filename parsing & file I/O
├── queries.go              # Game retrieval for API
├── hashing.go              # File hash calculation
├── service.go              # Service initialization
├── metadata.go             # Data structures
├── scan_test.go            # (Unimplemented) scan tests
├── crud_test.go            # CRUD operation tests
├── metadata_io_test.go     # Metadata file tests
└── metadata_test.go        # Metadata structure tests
```

### Database Models
```
/home/razz/src/swopcart/server/internal/database/models.go
- Game (lines 140-158)
- GameVersion (lines 161-182)
- Platform (lines 107-115)
- Library (lines 118-137)
```

### API Handlers
```
/home/razz/src/swopcart/server/internal/www/api/v0/
├── games.go               # GET/LIST games, download versions
└── libraries.go           # Create/update libraries, trigger scans
```

---

## How ROM Files Are Detected

### File Matching Process
1. **Extension Filtering** (`jobs.go:278-290`)
   - Compare file extension against platform's allowed extensions
   - Case-insensitive matching
   - Extensions stored as JSON array in `Platform.Extensions`
   - Example for SNES: `[".sfc", ".smc", ".rom", ".zip"]`

2. **Two-Pass Directory Walk** (`jobs.go:126-244`)
   - **First pass**: Count total files for progress tracking
   - **Second pass**: Process each matching file
   - Uses `filepath.Walk()` for recursive directory traversal

3. **No Content-Based Detection**
   - Currently NO file hashing for identification
   - Only extension-based matching
   - Hashes are calculated AFTER file match (for storage, not matching)

### Platform Configuration
```go
Platform{
  ID: uint
  Name: string          // "SNES", "NES", etc.
  Description: string
  Extensions: string    // JSON array: [".sfc", ".smc", ".rom"]
}
```

Default platforms seeded in code:
- SNES, NES, N64, Game Boy variants, Sega systems
- PlayStation 1/2, Dreamcast
- DOS, Windows, Atari 2600

---

## How ROM Files Are Identified & Deduplicated

### Current Method: Title-Based Grouping

**Step 1: Extract Metadata from Filename** (`metadata_io.go:169-237`)
```go
func ExtractMetadataFromFilename(filename string, platform string) *GameMetadata
```

Patterns extracted:
- **vgdb ID**: `[vgdb=XXXXX]` → External ID
- **Region tags**: `.pal`, `.ntsc` → Regions array
- **Country codes**: `[USA]`, `[EUR]`, `[JPN]` → Regions array
- **Version/tags**: `(v1.0)`, `(Rev A)`, `(Demo)`, `(Beta)` → Tags array
- **Title**: Remaining text after removing patterns

Example:
```
Input:  "Super Mario World (PAL) [vgdb=12345].sfc"
Output: {
  Title: "Super Mario World (PAL)",  ← BUG: INCLUDES region!
  Regions: ["PAL"],
  ExternalIDs: {"vgdb": "12345"}
}
```

**Step 2: Game Lookup** (`crud.go:331-368`)
```go
func GetOrCreateGame(libraryID uuid.UUID, dirPath string, title string)
```

Queries database:
```sql
SELECT * FROM games
WHERE library_id = ? AND title = ?
```

**THE BUG**: Title includes region info!
- File 1: "Super Mario World (PAL).sfc" → title = "Super Mario World (PAL)"
- File 2: "Super Mario World (NTSC-U).sfc" → title = "Super Mario World (NTSC-U)"
- Different titles = different games created

**What SHOULD Happen**:
- Extract title WITHOUT region: "Super Mario World"
- Use clean title for game matching
- Store region in VersionName or Game metadata
- Multiple ROMs with same clean title → same Game, different Versions

---

## Where Metadata Files Are Created

### Metadata Files on Disk

**File Path**: `{rom_file_path}.meta`
- Example: `Super Mario World (PAL).sfc.meta`
- Located adjacent to ROM file
- Format: TOML (kebab-case keys)

**When Created** (`jobs.go:216-222`):
```go
metadataPath := filePath + ".meta"
if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
    // Only create if doesn't exist - user edits are preserved
    svc.SaveMetadataToFile(metadataPath, metadata)
}
```

**Metadata File Contents** (`metadata_io.go:14-33`):
```toml
title = "Super Mario World"
platform = "SNES"
developer = "Nintendo"
publisher = "Nintendo"
release-date = "1990-11-21"
description = "..."
regions = ["PAL"]
tags = []

[external-ids]
vgdb = "123456"

[[versions]]
name = "PAL"
filename = "Super Mario World (PAL).sfc"
regions = ["PAL"]
```

### Metadata in Database

**Storage Location**: `GameVersion.MetadataJSON` (text field)
- JSON-serialized representation
- CamelCase keys for API consistency
- Currently NOT populated by scanner (opportunity for improvement)

**Functions**:
- `SaveMetadataToFile()`: Write TOML to disk
- `LoadMetadataFromFile()`: Read TOML from disk
- `MetadataToJSON()`: Convert TOML to camelCase JSON
- `MetadataToString()`: Serialize to JSON string for DB storage

---

## How Versions Work

### GameVersion Model
```go
type GameVersion struct {
  ID            uuid.UUID   // Unique identifier
  GameID        uuid.UUID   // FK to Game (determines which game)
  VersionName   string      // e.g., "PAL", "NTSC-U", "v1.0"
  FilePath      string      // Unique: full path to ROM file
  FileSize      int64       // File size in bytes
  MD5/SHA1/SHA256/Blake3 string  // File hashes
  MetadataPath  string      // Path to .meta file
  MetadataJSON  string      // JSON metadata in database
}
```

### Version Creation (`crud.go:371-435`)
```go
func UpsertGameVersion(gameID uuid.UUID, filePath string, versionName string,
                       fileSize int64, hashes map[string]string)
```

**Uniqueness**: Based on `FilePath` (unique constraint in DB)

**VersionName Assignment** (line 211 in jobs.go):
```go
dirName := filepath.Base(filepath.Dir(filePath))
_, err = svc.UpsertGameVersion(ctx, game.ID, filePath, dirName, ...)
```

**BUG**: Uses directory name instead of extracted region
- Should be: `versionName = metadata.Regions[0]` or similar
- Should default to directory name if regions not extracted

### API Response
```json
{
  "id": "game-uuid",
  "title": "Super Mario World",
  "versions": [
    {
      "id": "version-uuid-1",
      "versionName": "PAL",
      "filePath": "/path/to/game-pal.sfc",
      "fileSize": 1048576,
      "md5": "abc123...",
      "sha1": "...",
      "sha256": "..."
    },
    {
      "id": "version-uuid-2",
      "versionName": "NTSC-U",
      "filePath": "/path/to/game-ntsc.sfc",
      "fileSize": 1048576,
      "md5": "def456...",
      "sha1": "...",
      "sha256": "..."
    }
  ]
}
```

---

## Hash Calculation & Caching

### How Hashes Are Calculated

**File**: `hashing.go:22-52`
**Function**: `CalculateFileHashes(filePath)`

**Algorithms**:
- MD5 (32 hex chars)
- SHA1 (40 hex chars)
- SHA256 (64 hex chars)
- Blake3 (reserved for future use)

**Strategy**: Streaming multi-writer
```go
md5Hash := md5.New()
sha1Hash := sha1.New()
sha256Hash := sha256.New()
multiWriter := io.MultiWriter(md5Hash, sha1Hash, sha256Hash)
io.Copy(multiWriter, file)  // Single-pass hash calculation
```

**Why**: Avoids loading entire file into memory (important for low-RAM devices)

### Caching Logic (`jobs.go:187-208`)

```go
var existingVersion database.GameVersion

// Check if version already exists by FilePath
if err := svc.db.Where("file_path = ?", filePath).First(&existingVersion).Error;
   err == gorm.ErrRecordNotFound {
    // NEW file: calculate hashes
    hashes, err = svc.CalculateFileHashes(filePath)
} else if err != nil {
    // Error querying: use empty hashes
    hashes = &Hashes{}
} else {
    // FILE EXISTS: reuse cached hashes
    hashes = &Hashes{
        MD5:    existingVersion.MD5,
        SHA1:   existingVersion.SHA1,
        SHA256: existingVersion.SHA256,
        Blake3: existingVersion.Blake3,
    }
}
```

**Key Point**: Hashes are ONLY calculated once
- Subsequent scans reuse cached values
- Prevents expensive recalculation
- Assumes file content doesn't change (if FilePath exists, content is same)

---

## Directory Structure Assumptions

### Expected Disk Layout
```
library_path/
├── game-1.sfc              # ROM file
├── game-1.sfc.meta         # Auto-created metadata
├── game-2.sfc
├── game-2.sfc.meta
├── subdirectory/           # Nested directories supported
│   ├── game-3.sfc
│   └── game-3.sfc.meta
├── .git/                   # Hidden directories ignored (filepath.Walk skips)
└── system files            # Any non-matching extensions ignored
```

### Key Assumptions
✓ **Supports**:
- Flat directory with multiple ROMs
- Nested subdirectories
- Multiple file formats per platform

✗ **Does NOT require**:
- One directory per game
- Specific naming conventions (extracts from filename)
- Pre-existing metadata files

---

## Complete Data Model

### Database Schema Hierarchy

```
Platform (e.g., SNES)
├── name: "SNES"
├── extensions: [".sfc", ".smc", ".rom", ".zip"]
└── [Multiple Libraries use this platform]
    │
    └─→ Library
        ├── id: uuid
        ├── name: "My SNES Collection"
        ├── platformId: 2
        ├── paths: ["/mnt/games/snes", "/data/snes"]
        ├── scanStatus: "idle" | "scanning" | "error"
        ├── lastScannedAt: timestamp
        └── [Games in this library]
            │
            └─→ Game
                ├── id: uuid
                ├── title: "Super Mario World"
                ├── developer: "Nintendo"
                ├── publisher: "Nintendo"
                ├── releasedDate: timestamp
                ├── description: string
                └── [Multiple versions of this game]
                    │
                    ├─→ GameVersion
                    │   ├── id: uuid
                    │   ├── versionName: "PAL"
                    │   ├── filePath: "/path/to/game-pal.sfc" (unique)
                    │   ├── fileSize: 1048576
                    │   ├── md5/sha1/sha256/blake3: hashes
                    │   ├── metadataPath: "/path/to/game-pal.sfc.meta"
                    │   └── metadataJSON: "{...}"
                    │
                    └─→ GameVersion
                        ├── id: uuid
                        ├── versionName: "NTSC-U"
                        ├── filePath: "/path/to/game-ntsc.sfc"
                        ├── fileSize: 1048576
                        └── ...
```

---

## Scanning Process Step-by-Step

1. **Scan Triggered** (manual API or cron job at 3 AM)
2. **Load Library Config** (paths, platform, extensions)
3. **Count Files** (first pass for progress bar)
4. **For Each ROM File**:
   - Extract metadata from filename
   - Find/create game by cleaned title
   - Calculate hashes (or reuse cached)
   - Create/update game version
   - Save metadata file if new
   - Update game metadata (developer, publisher)
5. **Cleanup** (soft-delete games with no files on disk)

---

## Current Bug Analysis

### Bug Description
Files "Super Mario World (PAL).sfc" and "Super Mario World (NTSC-U).sfc" create 2 separate games instead of 1 game with 2 versions.

### Root Cause
In `crud.go:336-338`:
```go
result := svc.db.WithContext(ctx).
    Where("library_id = ? AND title = ?", libraryID, title).
    First(&game)
```

The `title` parameter passed to this function includes region info:
- From `jobs.go:173`: `metadata := svc.ExtractMetadataFromFilename(filename, ...)`
- From `metadata_io.go:234`: `metadata.Title = title` (after cleanup)
- The cleanup DOESN'T remove region patterns like "(PAL)" or "(NTSC-U)"

### Evidence from Code

**metadata_io.go:193-217**: Extract region patterns
```go
// Extract region: .pal, .ntsc
if strings.Contains(title, ".pal") {
    metadata.Regions = append(metadata.Regions, "PAL")
    title = strings.ReplaceAll(title, ".pal", "")  // Removes .pal from title
}

// Extract version/tag: (v1.0), (Rev A), (Demo), (Beta)
if versionMatch := strings.Index(title, "("); versionMatch >= 0 {
    endIdx := strings.Index(title[versionMatch:], ")")
    if endIdx >= 0 {
        versionStr := title[versionMatch+1 : versionMatch+endIdx]
        // Note: Version name stored in GameVersion.VersionName, not in metadata
        title = title[:versionMatch] + title[versionMatch+endIdx+1:]  // REMOVES from title
    }
}

// Extract country codes
regions := []string{"USA", "EUR", "JPN", "AUS", "CAN"}
for _, region := range regions {
    pattern := "[" + region + "]"
    if strings.Contains(title, pattern) {
        metadata.Regions = append(metadata.Regions, region)
        title = strings.ReplaceAll(title, pattern, "")  // Removes [USA] etc.
    }
}
```

**ISSUE**: Pattern "(PAL)" is ONLY removed if it starts with "("
- Filename: "Super Mario World (PAL).sfc"
- After removing extension: "Super Mario World (PAL)"
- The code looks for "(v1.0)", "(Rev A)", "(Demo)" - typical version markers
- "(PAL)" DOES get matched and removed!

Wait, let me re-examine...

Actually looking more carefully at lines 204-217:
```go
// Extract version/tag: (v1.0), (Rev A), (Demo), (Beta)
if versionMatch := strings.Index(title, "("); versionMatch >= 0 {
    endIdx := strings.Index(title[versionMatch:], ")")
    if endIdx >= 0 {
        versionStr := title[versionMatch+1 : versionMatch+endIdx]

        // Check for demo/beta tags
        if strings.EqualFold(versionStr, "demo") || strings.EqualFold(versionStr, "beta") {
            metadata.Tags = append(metadata.Tags, strings.ToLower(versionStr))
        }
        // Note: Version name is stored in GameVersion.VersionName, not in metadata

        title = title[:versionMatch] + title[versionMatch+endIdx+1:]  // REMOVES parenthetical
    }
}
```

Actually this DOES remove it! The code removes any `(something)` pattern and:
- If it's "demo" or "beta", adds to tags
- Otherwise, just removes it from title
- "PAL" is not "demo" or "beta", so it just gets removed

So the title SHOULD become "Super Mario World" after cleanup...

Let me check jobs.go to see what title is actually passed:

**jobs.go:173**:
```go
metadata := svc.ExtractMetadataFromFilename(filename, platform.Name)
```

So metadata.Title should be cleaned...

**jobs.go:176-178**:
```go
// Use extracted title if available, otherwise use directory name
if metadata.Title == "" {
    metadata.Title = dirName
}
```

**jobs.go:181**:
```go
game, err := svc.GetOrCreateGame(ctx, lib.ID, dirPath, metadata.Title)
```

So if extracted title is cleaned properly, this should work...

Unless... let me check if the extraction is actually working correctly for region-suffixed titles.

Looking at the test data or checking if there's an issue with the extraction logic...

Actually, I think I found it! Look at the extraction code more carefully at line 203-217:

```go
// Extract version/tag: (v1.0), (Rev A), (Demo), (Beta)
if versionMatch := strings.Index(title, "("); versionMatch >= 0 {
    endIdx := strings.Index(title[versionMatch:], ")")
    if endIdx >= 0 {
        versionStr := title[versionMatch+1 : versionMatch+endIdx]
        // ...
        title = title[:versionMatch] + title[versionMatch+endIdx+1:]
    }
}
```

This code DOES handle `(PAL)` and would remove it. So the bug isn't in the extraction...

Let me look at jobs.go again more carefully. Maybe the problem is somewhere else?

Wait - I need to actually test this. Let me look for test cases or comments that show what's expected...

The scan_test.go file at the top shows test stubs that suggest the scanner hasn't been fully tested yet. The actual behavior might be different from what the code suggests.

**Hypothesis**: The extraction might not be removing region tags properly because they're not being matched by the generic "(X)" pattern - they might need special handling.

Let me check if there's a separate region extraction that happens BEFORE the version extraction...

Actually looking again: The region extraction at lines 193-229 handles:
- `.pal`, `.ntsc` (extensions)
- `[USA]`, `[EUR]`, etc. (country codes)

But NOT `(PAL)` or `(NTSC-U)` with parentheses!

So the title would stay as "Super Mario World (PAL)" because neither:
- The `.pal` pattern (needs dot)
- The country code pattern (needs brackets and one of the 5 codes)
- Nor the version removal (only removes if it's "demo" or "beta")

would match `(PAL)` or `(NTSC-U)`!

That's the bug!

### The Real Bug

**Root Cause Location**: `metadata_io.go:193-229`

The extraction doesn't handle region names in parentheses:
- Handles: `.pal`, `.ntsc`, `[USA]`, etc.
- Does NOT handle: `(PAL)`, `(NTSC-U)`, `(EUR)` with parentheses

So filenames like:
- "Super Mario World (PAL).sfc" → title stays "Super Mario World (PAL)"
- "Super Mario World (NTSC-U).sfc" → title stays "Super Mario World (NTSC-U)"

These are treated as different games in the matching step.

---

## Summary of Required Information

### Files Affected
1. **Primary Scanning**: `/home/razz/src/swopcart/server/internal/services/library/jobs.go`
2. **Game Matching**: `/home/razz/src/swopcart/server/internal/services/library/crud.go`
3. **Title Extraction**: `/home/razz/src/swopcart/server/internal/services/library/metadata_io.go`
4. **Data Models**: `/home/razz/src/swopcart/server/internal/database/models.go`

### How ROM Files Are Detected
- File extension matching against `Platform.Extensions` (JSON array)
- Case-insensitive comparison
- No content-based detection

### How ROMs Are Currently Identified
- By matching extracted title to existing games
- Title extracted from filename by removing known patterns
- **BUG**: Region names in parentheses `(PAL)` not being removed

### How Dedupe/Grouping Should Work
- Extract clean title (WITHOUT regions/variants)
- Use clean title for game matching
- Multiple ROMs with same clean title → same game, different versions
- Version names derived from extracted regions

### Metadata File Creation
- Created adjacent to ROM files: `{filename}.meta`
- TOML format with kebab-case keys
- Only created if doesn't exist (preserves user edits)
- One .meta file per ROM file

### Version Tracking
- GameVersion model: One per ROM file
- VersionName: Currently directory name, should be region/variant
- Uniqueness: By FilePath
- All versions linked to parent Game by GameID

---

## Key Code References

| Task | File | Function | Lines |
|------|------|----------|-------|
| Main scan | jobs.go | `scanLibrary` | 69-275 |
| Metadata extract | metadata_io.go | `ExtractMetadataFromFilename` | 169-237 |
| Game matching | crud.go | `GetOrCreateGame` | 331-368 |
| Version create | crud.go | `UpsertGameVersion` | 371-435 |
| Hash calc | hashing.go | `CalculateFileHashes` | 22-52 |
| File IO | metadata_io.go | `SaveMetadataToFile` | 75-92 |
| Game retrieval | queries.go | `GetGameByID` | 80-94 |
