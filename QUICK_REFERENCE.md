# Game Library Scanner - Quick Reference

## 🎯 The Bug in One Sentence
Files with region variants in parentheses like "Game (PAL).sfc" and "Game (NTSC-U).sfc" create 2 separate games instead of 1 game with 2 versions.

## 🔍 Root Cause
`metadata_io.go:193-229` extracts region patterns (`.pal`, `[USA]`, `(Demo)`) from filenames but DOESN'T handle parenthetical regions like `(PAL)` or `(NTSC-U)`, so they stay in the title. Different titles = different games.

## 📁 Critical Files

| File | Purpose | Issue |
|------|---------|-------|
| `jobs.go` | Scanning orchestrator | Passes uncleaned title to game matching |
| `metadata_io.go` | Title extraction | Doesn't remove `(REGION)` patterns |
| `crud.go` | Game matching & creation | Uses title as unique key, doesn't handle variants |
| `models.go` | Database schema | Game/GameVersion relationship design |

## 🔄 Data Flow

```
ROM File → Extract Metadata → Find/Create Game → Create Version → Save Metadata
             (dirty title)    (title is key)    (for each ROM)    (to disk)
                                      ↓
                          "Mario (PAL)" ≠ "Mario (NTSC-U)"
                               = 2 games (BUG!)
```

## 📊 Data Structure

```
Game (represents the title)
  └─ Versions (different ROM files of same game)
     ├─ Version 1: PAL variant
     └─ Version 2: NTSC-U variant
```

Currently broken:
- Game: "Mario (PAL)"
  └─ Version 1: (PAL) file

- Game: "Mario (NTSC-U)"
  └─ Version 1: (NTSC-U) file

## 🛠️ What Needs to Change

### 1. Extract Clean Title
Remove ALL region/variant markers from title BEFORE using it for game matching:
- Region extensions: `.pal`, `.ntsc`
- Country codes: `[USA]`, `[EUR]`, `[JPN]`
- **Region in parens: `(PAL)`, `(NTSC-U)`, `(EUR)`, `(JPN)`, `(USA)`** ← MISSING
- Version markers: `(v1.0)`, `(Rev A)`, `(Demo)`, `(Beta)`

### 2. Use Extracted Region for VersionName
Currently `VersionName = dirName` (directory)
Should be: `VersionName = metadata.Regions[0]` or fallback to dirName

### 3. Game Matching Strategy
Query should use:
```go
Where("library_id = ? AND title = ?", libraryID, cleanedTitle)
```

Not:
```go
Where("library_id = ? AND title = ?", libraryID, titleWithRegion)
```

## 🗂️ File Paths Summary

```
/home/razz/src/swopcart/server/

Internal Services:
  internal/services/library/
  ├── jobs.go                 ← Scanning orchestrator
  ├── crud.go                 ← Game/Version CRUD ops
  ├── metadata_io.go          ← Filename parsing (HAS THE BUG)
  ├── queries.go              ← Game retrieval
  ├── hashing.go              ← Hash calculation
  └── service.go              ← Initialization

Database:
  internal/database/
  └── models.go               ← Game/GameVersion schema

API:
  internal/www/api/v0/
  ├── games.go                ← Game endpoints
  └── libraries.go            ← Library endpoints
```

## 🔑 Key Functions

| Function | File | Line | Purpose |
|----------|------|------|---------|
| `scanLibrary` | jobs.go | 69 | Main scan orchestrator |
| `ExtractMetadataFromFilename` | metadata_io.go | 169 | Parse filename (HAS BUG) |
| `GetOrCreateGame` | crud.go | 331 | Find/create game by title |
| `UpsertGameVersion` | crud.go | 371 | Create/update version |
| `CalculateFileHashes` | hashing.go | 22 | Hash calculation |
| `SaveMetadataToFile` | metadata_io.go | 75 | Write .meta file |

## 🎮 ROM Detection

**Method**: File extension matching
- Extensions stored as JSON array in `Platform.Extensions`
- Case-insensitive comparison
- Checked: `[".sfc", ".smc", ".rom", ".zip"]` etc.
- No content-based detection (no hash matching for ID)

## 💾 Metadata Storage

### Disk Files (.meta)
- Path: `{rom_file}.meta` (e.g., `game.sfc.meta`)
- Format: TOML (kebab-case)
- One per ROM file
- Created if not exists (preserves edits)

### Database (GameVersion.MetadataJSON)
- Format: JSON (camelCase)
- Currently NOT populated by scanner
- One field per GameVersion

## 🚀 How Versions Work

```
Game{
  ID: uuid
  Title: "Super Mario World"  ← Cleaned, without regions

  Versions: [
    GameVersion{
      ID: uuid
      VersionName: "PAL"       ← Region from metadata
      FilePath: "/path/game-pal.sfc"
      Hashes: { md5, sha1, ... }
    },
    GameVersion{
      ID: uuid
      VersionName: "NTSC-U"    ← Region from metadata
      FilePath: "/path/game-ntsc.sfc"
      Hashes: { md5, sha1, ... }
    }
  ]
}
```

## ⚡ Quick Facts

- **Scan triggered**: Manual API call or 3 AM cron job
- **Directory traversal**: Recursive with filepath.Walk
- **Hash calculation**: Streaming (MD5, SHA1, SHA256)
- **Hash caching**: Reused on subsequent scans
- **Game deletion**: Soft-deleted when no versions on disk
- **Title matching**: Currently broken - includes region info
- **VersionName source**: Currently directory name (should be region)

## 📋 Checklist for Fix

- [ ] Add region parentheses extraction: `(PAL)`, `(NTSC-U)`, etc.
- [ ] Create cleaned title without regions
- [ ] Use cleaned title in `GetOrCreateGame` query
- [ ] Derive `VersionName` from extracted regions
- [ ] Update `jobs.go` to pass extracted regions to version creation
- [ ] Write tests for new title extraction logic
- [ ] Test with multiple region variants
- [ ] Verify soft-deletion still works correctly
