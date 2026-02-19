# 🎮 Game Library Service - Implementation Plan

**Status**: Phase 1 - Database & Platform Setup (IN PROGRESS)
**Last Updated**: 2026-02-20
**Created**: 2026-02-20

---

## 📋 Final Design Summary - All Decisions Locked

### Core Design Decisions

| Decision | Choice |
|----------|--------|
| **Platform Data** | `platforms.toml` in data folder, hardcoded defaults, admin-editable |
| **Path Validation** | Must exist and be readable at library creation time |
| **Concurrent Scans** | Only one scan per library at a time (prevent race conditions) |
| **File Errors** | Log and continue (no TOCTOU checks, trust filesystem state) |
| **Metadata Format** | TOML on disk (kebab-case), JSON in API (camelCase) |
| **Metadata Updates** | Update both file and database simultaneously |
| **Game Organization** | One Game per directory; versions tracked as GameVersion records |
| **File Filtering** | Scan only files matching platform's extensions |
| **Scan Schedule** | Daily (default: 3 AM), **PLUS** triggered on library creation |
| **Progress UI** | Basic status only; real-time progress deferred to future phase |

---

## 🗂️ Database Schema

### Platform
```go
type Platform struct {
    ID          uint      `gorm:"primaryKey"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt

    Name        string           `gorm:"uniqueIndex"` // "NES", "SNES", etc.
    Description string
    Extensions  datatypes.JSONSlice `gorm:"type:jsonb"` // [".nes", ".rom", ".zip"]
}
```

### Library
```go
type Library struct {
    ID          uuid.UUID `gorm:"primaryKey"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt

    Name        string
    Description string
    PlatformID  uint
    Platform    Platform `gorm:"foreignKey:PlatformID"`

    Paths               datatypes.JSONSlice `gorm:"type:jsonb"` // ["/mnt/games/nes"]
    LastScannedAt       *time.Time
    ScanStatus          string // "idle", "scanning", "error"
    LastScanError       string
    CurrentScanJobID    *uuid.UUID // Track active scan to prevent concurrent scans
}
```

### Game
```go
type Game struct {
    ID        uuid.UUID `gorm:"primaryKey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt

    LibraryID   uuid.UUID
    Library     Library `gorm:"foreignKey:LibraryID"`

    Title       string `gorm:"index"` // Clean title from filename
    PlatformID  uint
    Platform    Platform `gorm:"foreignKey:PlatformID"`

    Developer   string
    Publisher   string
    ReleasedDate *time.Time
    Description string

    Versions    []GameVersion `gorm:"foreignKey:GameID"`
}
```

### GameVersion
```go
type GameVersion struct {
    ID        uuid.UUID `gorm:"primaryKey"`
    CreatedAt time.Time
    UpdatedAt time.Time

    GameID       uuid.UUID
    Game         Game `gorm:"foreignKey:GameID"`

    VersionName  string // "1.0", "PAL", "NTSC", "Demo", etc.
    FilePath     string `gorm:"uniqueIndex"` // Full path to file
    FileSize     int64

    // Hashes calculated on first scan only
    MD5      string // 32 hex chars
    SHA1     string // 40 hex chars
    SHA256   string // 64 hex chars
    Blake3   string // 64 hex chars

    MetadataPath string // Path to .meta file

    // JSON representation of TOML metadata for API responses
    MetadataJSON datatypes.JSONType `gorm:"type:jsonb"`
}
```

---

## 📁 File Organization Patterns

### Pattern A: Flat Files
```
/mnt/games/nes/
├── zelda.rom
├── zelda.rom.meta          # TOML with kebab-case keys
├── mario.rom
├── mario.rom.meta
```

### Pattern B: Directories with Multi-Version Support
```
/mnt/games/nes/
├── zelda/
│   ├── zelda_1.0.rom
│   ├── zelda_pal.rom
│   └── meta.toml           # Parent metadata, tracks all versions
├── mario/
│   ├── mario.rom
│   └── meta.toml
```

---

## 📄 Metadata TOML Schema

**Location**: `game.rom.meta` or `game_dir/meta.toml`

**File Format** (kebab-case keys):
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

**JSON API Response** (camelCase):
```json
{
  "title": "The Legend of Zelda",
  "platform": "NES",
  "developer": "Nintendo",
  "publisher": "Nintendo",
  "releaseDate": "1986-07-27",
  "description": "Epic adventure game",
  "externalIds": {"vgdb": "12345", "igdb": "67890"},
  "regions": ["NTSC", "PAL"],
  "tags": ["action", "adventure", "classic"],
  "versions": [...]
}
```

---

## 🔑 Filename Parsing Rules

Extract and remove patterns from filename to get clean title:

- `[vgdb=XXXXX]` → `externalIds.vgdb = "XXXXX"`
- `.pal`, `.ntsc` → `regions = ["PAL"]` or `["NTSC"]`
- `(v1.0)`, `(Rev A)` → `versionName = "1.0"`
- `(Demo)`, `(Beta)` → `tags` + `versionName = "Demo"`
- `[USA]`, `[EUR]`, `[JPN]` → regions
- Remaining text (after cleanup) = `title`

---

## 📊 Platforms.toml Seed Data

**Location**: `$SWOPCART_DATA/platforms.toml`

Hardcoded with defaults, admins can modify to:
- Add new platforms
- Change extensions per platform
- Modify descriptions

Platforms included:
- NES, SNES, N64, Game Boy, Game Boy Color, Game Boy Advance
- Sega Genesis, Game Gear, Master System
- Atari 2600
- PlayStation 1, PlayStation 2, Dreamcast
- DOS, Windows

---

## 🔄 API Endpoints

### Library Management (Admin-only)
```
GET    /api/v0/libraries
POST   /api/v0/libraries
GET    /api/v0/libraries/:libraryId
PATCH  /api/v0/libraries/:libraryId
DELETE /api/v0/libraries/:libraryId
POST   /api/v0/libraries/:libraryId/scan
```

### Game Discovery (Authenticated)
```
GET    /api/v0/libraries/:libraryId/games?offset=0&limit=50&search=title
GET    /api/v0/games/:gameId
GET    /api/v0/games/:gameId/versions/:versionId/download
PATCH  /api/v0/games/:gameId (admin-only)
```

---

## 🚀 Implementation Phases

### Phase 1: Database & Platform Setup
- [ ] Add Platform, Library, Game, GameVersion models
- [ ] Create platforms.toml seed file
- [ ] Implement platform loading on service init
- [ ] GORM auto-migrations

### Phase 2: LibraryService Core
- [ ] CreateLibrary() - with path validation
- [ ] UpdateLibrary() - allow path changes if they exist
- [ ] DeleteLibrary() - soft delete
- [ ] ListLibraries() - with counts
- [ ] GetLibrary() - with scan status
- [ ] Metadata I/O (TOML ↔ JSON)
- [ ] Filename parsing
- [ ] Hash calculation (first scan only)

### Phase 3: Scanning Job
- [ ] library.scan job handler
- [ ] Path walking with extension filtering
- [ ] Concurrent scan prevention
- [ ] Directory vs flat file handling
- [ ] Progress reporting
- [ ] Auto-scan on library creation
- [ ] Daily scheduled scan (0 0 3 * * *)

### Phase 4: API Endpoints
- [ ] /api/v0/libraries/* endpoints
- [ ] /api/v0/games/* endpoints
- [ ] Download streaming with range requests
- [ ] Error codes & auth

### Phase 5: Service Integration
- [ ] Register LibraryService
- [ ] Register library.scan job
- [ ] End-to-end testing

### Phase 6: Frontend
- [ ] Type definitions & API client
- [ ] /library page (game browser)
- [ ] /settings/libraries page (admin)
- [ ] Game details & metadata modals

### Phase 7: Testing & Polish
- [ ] Unit tests
- [ ] Integration tests
- [ ] Edge cases
- [ ] UI refinements

---

## 🎯 Key Constraints

### Memory Efficiency (1GB RAM)
- Stream downloads, no buffering
- Hash calculation only on first scan
- Pagination (50 items/page)
- No real-time progress UI yet

### Data Integrity
- File is source of truth for metadata
- API updates → file + DB simultaneously
- Soft deletes for libraries and games
- Scan can be interrupted

### File Organization
- One Game per directory
- Versions tracked as GameVersion records
- Platform extensions used for filtering

---

## ⚠️ Error Codes

```
LibraryNotFound
LibraryAlreadyExists
PlatformNotFound
InvalidLibraryPath
PathNotFound
PathNotDirectory
ScanAlreadyInProgress
GameNotFound
VersionNotFound
CannotDownloadGame
```

---

## 📝 User Guide Notes

- Multiple games in a directory (except Mods, DLC, Versions) is not supported
- One Game per directory structure recommended
- Admins can edit `platforms.toml` to customize extensions per platform
- File metadata persists on disk; database mirrors it

---

## 🔗 Related Files

- `internal/database/models.go` - Database models
- `internal/services/library/` - Service implementation
- `internal/www/api/v0/` - API handlers
- `frontend/src/lib/api/` - Frontend client
- `frontend/src/pages/` - Frontend pages
- `data/platforms.toml` - Platform configuration (auto-created)
