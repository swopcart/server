# Swopcart Library - Key Findings & Critical Insights

## Critical Discovery: Two Metadata Systems

### 1. LEGACY Metadata (unused but tested)
**File**: `internal/services/library/metadata.go` (lines 63-88)

- Has `UUID` field (instance identifier)
- Versions as `map[string]Version` (keyed lookup)
- Per-version platform specification
- Per-version hashes and runner metadata
- **Used only in tests** (`metadata_test.go`)
- **Not integrated with scanner** (`jobs.go`)

### 2. ACTIVE Metadata (current scanner)
**File**: `internal/services/library/metadata_io.go` (lines 14-54)

- NO UUID field (generic game metadata)
- Versions as `[]VersionMetadata` (array)
- Single platform for entire game
- Only regions and filenames per version
- **Integrated with scanner** (`jobs.go`)
- **Tested in metadata_io_test.go**

**Question to resolve**: Should these be unified? The legacy Metadata struct has more features (UUID, RunnerMeta) but isn't used.

---

## Current Directory Traversal Approach

### Two-Pass Scan (jobs.go lines 126-245)

```go
// PASS 1: Count files (lines 126-145)
for _, scanPath := range paths {
    filepath.Walk(scanPath, func(filePath, info, err) error {
        if matchesExtension(filePath, extensions) {
            totalRecords++
        }
    })
}

// PASS 2: Process files (lines 148-245)
for _, scanPath := range paths {
    filepath.Walk(scanPath, func(filePath, info, err) error {
        // ... process each file
    })
}
```

**Why two passes?**
- First pass: Count total files for progress bar (totalRecords/processedRecords)
- Second pass: Actually create games and versions

**Performance impact**: Filesystem traversal x2 (could be optimized to single pass with pre-allocation)

### Key Variables Extracted per File

```go
dirPath := filepath.Dir(filePath)          // e.g., "/games/nes" or "/games/nes/zelda"
dirName := filepath.Base(dirPath)          // e.g., "nes" or "zelda"
filename := filepath.Base(filePath)        // e.g., "zelda.nes"

// Extracted metadata
metadata := svc.ExtractMetadataFromFilename(filename, platform.Name)
// Extracted title used for game matching
metadata.Title
// Directory name used for version name
dirName
```

---

## Game Matching Logic - The Root Cause of Version Splitting

### Current: Title-Only Matching (PROBLEMATIC)

```go
// From crud.go lines 331-368
result := svc.db.WithContext(ctx).
    Where("library_id = ? AND title = ?", libraryID, title).
    First(&game)
```

**Matching criteria**: Only `(LibraryID, Title)`

**Problem scenarios**:

1. **Different region variants**
   ```
   File: "Zelda (PAL).nes"
   File: "Zelda (NTSC).nes"

   Extracted titles:
   1. "Zelda" (PAL stripped)
   2. "Zelda" (NTSC stripped)

   Result: Same game if regions extracted cleanly!
   ```

2. **Version names in filename**
   ```
   File: "Zelda_v1.0.nes"
   File: "Zelda_v1.1.nes"

   Extracted titles:
   1. "Zelda_v1" (not matching (v1.0) pattern properly)
   2. "Zelda_v1" (not matching (v1.1) pattern properly)

   Result: SAME game, but may be intended as different versions!
   ```

3. **Current scanner behavior (ACTUAL BUG)**
   ```
   File: "Zelda_1.0.nes"
   File: "Zelda_PAL.nes"

   Extracted titles:
   1. "Zelda_1" (underscore preserved)
   2. "Zelda" (after pattern removal)

   Result: TWO DIFFERENT GAMES! ⚠️
   ```

### Metadata Fields NOT Used in Game Matching

These are extracted but ignored:
- `Regions` (["PAL"], ["NTSC"], etc.)
- `ExternalIDs` ({"vgdb": "12345"})
- `Tags` (["demo"], ["beta"])
- `Developer` / `Publisher`

**If we matched by Region too:**
```
// Enhanced matching (NOT IMPLEMENTED)
result := svc.db.WithContext(ctx).
    Where("library_id = ? AND title = ? AND regions = ?",
          libraryID, title, regions).
    First(&game)
```

This would create SEPARATE games for different regions, which may or may not be desired.

---

## Metadata File Naming Inconsistency

### Actual Current Practice
- Metadata saved to: `{filePath}.meta`
- Example: `/games/nes/zelda.nes` → `/games/nes/zelda.nes.meta`

### Documented Intended Practice (GAME_LIBRARY_PLAN.md)
- **Pattern A (Flat)**: `game-rom-file.meta`
- **Pattern B (Directory)**: Game directory with `meta.toml`

### Gap in Implementation
**jobs.go line 217:**
```go
metadataPath := filePath + ".meta"
if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
    if err := svc.SaveMetadataToFile(metadataPath, metadata); err != nil {
```

**Logic**: Always saves alongside ROM file, never looks for directory-level `meta.toml`

**Missing feature**: Directory-level metadata that covers multiple versions in one file

---

## Version Naming - Another Issue

### Current Implementation (jobs.go line 211)
```go
_, err = svc.UpsertGameVersion(ctx, game.ID, filePath, dirName, info.Size(), hashes.ToMap())
//                                                 ^^^^^^^
//                         Uses directory name as VersionName!
```

### Example Issues

```
Directory: /games/snes/zelda/

Files:
  ├── zelda_1.0.sfc      → VersionName: "zelda" (from dir)
  ├── zelda_pal.sfc      → VersionName: "zelda" (from dir)
  └── zelda_enhanced.sfc → VersionName: "zelda" (from dir)

Problem: All versions have SAME VersionName!
Expected:
  ├── zelda_1.0.sfc      → VersionName: "1.0"
  ├── zelda_pal.sfc      → VersionName: "PAL"
  └── zelda_enhanced.sfc → VersionName: "Enhanced"
```

### Missing Feature
The filename patterns are extracted but NOT stored:
- `(v1.0)` → extracted but discarded
- `(PAL)` → extracted but discarded
- `(Demo)` → extracted to Tags, not VersionName

---

## Metadata TOML Format - CURRENT vs INTENDED

### What ACTUALLY Gets Saved

**Current (per-file):**
```toml
title = "The Legend of Zelda"
platform = "NES"
developer = ""        # Extracted from filename (usually empty)
publisher = ""        # Extracted from filename (usually empty)
release-date = ""     # Never extracted from filename
description = ""      # Never extracted from filename

[external-ids]
vgdb = "12345"        # If [vgdb=12345] in filename

regions = ["PAL"]     # If .pal or [PAL] in filename
tags = ["demo"]       # If (Demo) in filename

# NOTE: versions section usually empty!
```

### What CAN Be Saved But Isn't Auto-Populated

```toml
[[versions]]
name = "1.0"
filename = "zelda_1.0.nes"
regions = ["NTSC"]
```

**Why?** The `versions` section in TOML should list all versions of a game, but:
1. Scanner only extracts from single file at a time
2. No logic to read back existing `versions` section and merge
3. Version discovery from filename patterns not stored

---

## Metadata Storage Locations

### Database: GameVersion.MetadataJSON

```go
type GameVersion struct {
    MetadataPath string  // e.g., "/games/nes/zelda.nes.meta"
    MetadataJSON *string // Serialized: '{"title":"...", ...}'
}
```

**When updated**:
- Neither is automatically updated during scan
- Must manually call UpdateGameMetadata + SaveMetadataToFile

### File System: .meta Files

```
/games/nes/
  ├── zelda.nes
  ├── zelda.nes.meta      # TOML, kebab-case, auto-created on first scan
  └── mario.nes.meta
```

**When updated**:
- Created only if doesn't exist (line 218: `os.IsNotExist`)
- Subsequent scans don't overwrite if file exists
- Manual updates via API should update both

---

## Platform Extensions - How They Work

### Storage
```go
type Platform struct {
    Extensions *string  // JSON string: "[".nes", ".rom", ".zip"]"
}
```

### Usage in Scanning

```go
// Get platform extensions (crud.go line 312)
var extensions []string
if platform.Extensions != nil {
    json.Unmarshal([]byte(*platform.Extensions), &extensions)
}

// Check each file (jobs.go line 161)
if !matchesExtension(filePath, extensions) {
    return nil
}

// Matching function (jobs.go line 278)
func matchesExtension(filePath string, extensions []string) bool {
    ext := strings.ToLower(filepath.Ext(filePath))
    for _, allowedExt := range extensions {
        if strings.EqualFold(ext, allowedExt) {
            return true
        }
    }
    return false
}
```

### Default Platforms Seeded

**service.go lines 111-131** seeds 15 platforms:
- NES, SNES, N64
- Game Boy (3 variants)
- Sega (3 variants)
- Atari 2600
- PlayStation (2 variants), Dreamcast
- DOS, Windows

**Missing support for**: Modern emulators (Switch, PS4), other systems

---

## Hash Calculation - Smart Caching

### When Hashes Are Calculated

```go
// jobs.go lines 187-208

// Check if version already exists
if err := svc.db.WithContext(ctx).Where("file_path = ?", filePath).First(&existingVersion).Error;
    err == gorm.ErrRecordNotFound {
    // NEW FILE: Calculate all hashes
    hashes, err = svc.CalculateFileHashes(filePath)
} else if err != nil {
    // ERROR: Use empty hashes
    hashes = &Hashes{}
} else {
    // EXISTS: Reuse existing hashes
    hashes = &Hashes{
        MD5:    existingVersion.MD5,
        SHA1:   existingVersion.SHA1,
        SHA256: existingVersion.SHA256,
        Blake3: existingVersion.Blake3,
    }
}
```

### Implementation Details

**hashing.go lines 22-52:**
- Stream reads file in 64KB chunks
- Calculates MD5, SHA1, SHA256 in parallel
- Returns struct with all four hashes
- Blake3 field reserved but not populated

**Performance**:
- First scan: Expensive (full file read + 4 hash calculations)
- Subsequent scans: Just DB lookup (fast)

---

## Soft Deletes - Games Not Files

### What Gets Soft-Deleted

```go
// jobs.go lines 247-271

for _, game := range allGames {
    var versions []database.GameVersion
    svc.db.WithContext(ctx).Where("game_id = ?", game.ID).Find(&versions)

    gameFound := false
    for _, version := range versions {
        if foundFiles[version.FilePath] {  // Is ANY version still on disk?
            gameFound = true
            break
        }
    }

    if !gameFound && game.DeletedAt.Time.IsZero() {
        // Game no longer exists on disk, soft-delete it
        svc.db.WithContext(ctx).Delete(&game)
    }
}
```

**Key point**: A Game is soft-deleted only if NONE of its versions exist on disk

**NOT soft-deleted**:
- GameVersion records (stays in DB forever with same FilePath)
- Library records (require explicit DELETE endpoint)

---

## Critical Limitations Summary

| Limitation | Impact | Workaround |
|-----------|--------|-----------|
| Game matching by title only | Multiple versions create duplicate games | Rename files to have same title |
| Version name from directory | All files in directory share version name | Manual metadata updates |
| No directory-level metadata | Can't share metadata across versions | Must update each file's .meta |
| No UUID in GameMetadata | Can't track instance identity | (Not needed for retro gaming) |
| Metadata creation one-time only | Filename changes don't update metadata | Manual re-scan or edit |
| Versions section never populated | TOML versions key unused | Future enhancement |
| No support for modern platforms | Hardcoded to retro consoles | Add to seedDefaultPlatforms() |
| Two-pass traversal | Slower on large libraries | Could optimize to single pass |

---

## Design Principles Observed

### Memory Efficiency (1GB RAM target)
✓ Streaming file hashing (not buffered)
✓ Pagination in API (50 items/page)
✓ One database transaction per scan (not per file)

### Fault Tolerance
✓ Errors on individual files don't stop scan
✓ Missing metadata files don't fail (created on-demand)
✓ Hash calculation errors use empty hashes (not failure)

### Idempotency
✓ Multiple scans safe (GameVersion.FilePath unique index)
✓ Missing .meta files not recreated (if exists, skip)
✓ Soft deletes preserve data (recover if file returns)

### API Design
✓ TOML kebab-case on disk (configuration standard)
✓ JSON camelCase in API (JavaScript convention)
✓ Automatic format conversion (not user responsibility)

---

## Recommendations for Enhancement

### High Priority
1. **Fix game title extraction** to not lose region/version info
2. **Extract version names from filenames** and store in VersionName
3. **Support directory-level metadata** (meta.toml in game directory)
4. **Implement metadata file updates** when TOML is manually edited

### Medium Priority
5. **Unify metadata systems** (merge legacy Metadata into GameMetadata)
6. **Cache MetadataJSON per version** (avoid re-serialization on API calls)
7. **Add version merge/unmerge** (combine/split multiple files under one game)

### Low Priority
8. **Optimize to single-pass scanning** (combine counting and processing)
9. **Add more modern platforms** (Switch, PS4, mobile)
10. **Implement real-time progress UI** (WebSocket updates)
