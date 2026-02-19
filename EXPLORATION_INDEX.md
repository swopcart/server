# Game Library Scanner Exploration - Complete Index

This is the index for all exploration documents. Start here!

## 📄 Documents Created

### 1. **QUICK_REFERENCE.md** ⭐ START HERE
   - **Purpose**: One-page summary of the bug, root cause, and fix checklist
   - **Best for**: Quick understanding, implementation checklist
   - **Read time**: 5 minutes

### 2. **SCANNER_EXPLORATION_SUMMARY.md**
   - **Purpose**: Comprehensive deep-dive with all technical details
   - **Best for**: Complete understanding, architectural decisions
   - **Covers**: File locations, data models, processes, API responses, bug analysis
   - **Read time**: 20-30 minutes

### 3. **SCANNER_ANALYSIS.md**
   - **Purpose**: Detailed analysis of each scanning stage
   - **Best for**: Understanding specific components (detection, hashing, metadata)
   - **Covers**: ROM detection, deduplication, hashing, versions, metadata
   - **Read time**: 15-20 minutes

### 4. **SCANNER_FLOW.md**
   - **Purpose**: Visual data flow diagrams showing the bug
   - **Best for**: Visual learners, understanding bug manifestation
   - **Covers**: ASCII flow diagrams, bug demonstration, data transformations
   - **Read time**: 10-15 minutes

## 🎯 Quick Navigation

**I want to...**

- **Fix the bug immediately** → Read `QUICK_REFERENCE.md` (5 min) → Implement fix
- **Understand the architecture** → Read `SCANNER_EXPLORATION_SUMMARY.md` (30 min)
- **See how files are detected** → Go to Section "How ROM Files Are Detected" in any doc
- **Understand the bug** → Read `QUICK_REFERENCE.md` "Root Cause" + `SCANNER_FLOW.md` "The Bug"
- **Find critical code locations** → See "Key Code References" table in any doc
- **Understand data models** → Go to "Data Model Structure" in `SCANNER_EXPLORATION_SUMMARY.md`
- **See the scanning process** → Go to "Scanning Process Step-by-Step" in any doc
- **Learn about metadata** → Go to "Where Metadata Files Are Created" in any doc

## 📊 Quick Fact Reference

| Fact | Value |
|------|-------|
| **Main scan file** | `/home/razz/src/swopcart/server/internal/services/library/jobs.go` |
| **Bug location** | `metadata_io.go:193-229` (ExtractMetadataFromFilename) |
| **Game matching** | `crud.go:331-368` (GetOrCreateGame) |
| **Version creation** | `crud.go:371-435` (UpsertGameVersion) |
| **Bug description** | Region variants create separate games instead of versions |
| **ROM detection** | File extension matching (case-insensitive) |
| **Hash algorithms** | MD5, SHA1, SHA256, Blake3 (reserved) |
| **Hash strategy** | Streaming multi-writer, cached on subsequent scans |
| **Metadata format** | TOML on disk (kebab-case), JSON in DB (camelCase) |
| **Metadata location** | `{rom_file}.meta` adjacent to each ROM |
| **Versions per game** | Multiple (one per ROM file with same cleaned title) |
| **Scan frequency** | 3 AM daily (cron: `0 0 3 * * *`) + manual trigger |
| **Directory structure** | Flat or nested, recursive walk with filepath.Walk |

## 📁 File Structure Overview

```
/home/razz/src/swopcart/server/

Documentation (NEW):
├── EXPLORATION_INDEX.md          ← You are here
├── QUICK_REFERENCE.md            ← Start here for quick understanding
├── SCANNER_EXPLORATION_SUMMARY.md ← Deep dive
├── SCANNER_ANALYSIS.md           ← Component analysis
└── SCANNER_FLOW.md               ← Visual diagrams

Implementation (EXISTING):
internal/
├── services/library/
│   ├── jobs.go                   ← Scanning orchestrator (main entry: scanLibrary)
│   ├── crud.go                   ← Game/Version CRUD (bug area: GetOrCreateGame)
│   ├── metadata_io.go            ← Filename parsing (BUG HERE: ExtractMetadataFromFilename)
│   ├── queries.go                ← Game retrieval for API
│   ├── hashing.go                ← Hash calculation
│   ├── service.go                ← Service initialization
│   ├── metadata.go               ← Data structures
│   ├── scan_test.go              ← (Unimplemented) scan tests
│   ├── crud_test.go              ← CRUD tests
│   ├── metadata_io_test.go       ← Metadata file tests
│   └── metadata_test.go          ← Metadata structure tests
│
├── database/
│   └── models.go                 ← Database schemas (Game, GameVersion, etc.)
│
└── www/api/v0/
    ├── games.go                  ← Game API endpoints
    └── libraries.go              ← Library API endpoints
```

## 🔍 The Bug at a Glance

**Symptom**: "Super Mario World (PAL).sfc" + "Super Mario World (NTSC-U).sfc" = 2 games ❌

**Root Cause**: Title extraction doesn't remove region names in parentheses

**Location**: `metadata_io.go:193-229` in `ExtractMetadataFromFilename()`

**Current extraction**:
- ✓ Removes: `.pal`, `.ntsc`, `[USA]`, `[EUR]`, `(Demo)`, `(Beta)`
- ✗ Doesn't remove: `(PAL)`, `(NTSC-U)`, `(EUR)`, `(JPN)`, `(USA)`

**Result**:
- File 1 title: "Super Mario World (PAL)"
- File 2 title: "Super Mario World (NTSC-U)"
- Different titles → different games (BUG)

**Expected**:
- Both become: "Super Mario World"
- Same title → same game with 2 versions ✓

## 🎮 Database Schema Quick View

```
Platform (e.g., SNES)
└─ Library (e.g., "My SNES Games")
   └─ Game (e.g., "Super Mario World")
      ├─ GameVersion (PAL variant)
      │  └─ FilePath: game-pal.sfc
      │  └─ VersionName: "PAL"
      │  └─ Hashes: MD5, SHA1, SHA256
      └─ GameVersion (NTSC-U variant)
         └─ FilePath: game-ntsc.sfc
         └─ VersionName: "NTSC-U"
         └─ Hashes: MD5, SHA1, SHA256
```

## 🔧 Implementation Details by Component

### ROM Detection
- **File**: jobs.go + service.go
- **Method**: Extension matching against Platform.Extensions
- **Extensions**: Stored as JSON array per platform
- **Case**: Insensitive comparison
- **Content**: No hash-based detection (only extension)

### Title Extraction
- **File**: metadata_io.go
- **Function**: ExtractMetadataFromFilename()
- **Current behavior**: Removes some patterns but not `(REGION)`
- **Result**: Title still contains region info → wrong game matching

### Game Matching
- **File**: crud.go
- **Function**: GetOrCreateGame()
- **Strategy**: Query by (libraryID + title)
- **Issue**: Title includes region → different games created

### Version Creation
- **File**: crud.go
- **Function**: UpsertGameVersion()
- **Uniqueness**: FilePath (one version per ROM file)
- **VersionName**: Currently dirName, should be extracted region

### Hash Calculation
- **File**: hashing.go
- **Algorithms**: MD5, SHA1, SHA256 (streaming, single-pass)
- **Strategy**: Only calculated on first scan, cached on subsequent
- **Benefit**: Saves memory and time on low-resource devices

### Metadata Files
- **Location**: `{rom_file}.meta` (adjacent to ROM)
- **Format**: TOML (kebab-case keys)
- **When created**: Only if doesn't exist (preserves user edits)
- **One per**: ROM file (not per game)

## 🚀 Scanning Process Overview

1. Scan triggered (API or cron 3 AM)
2. Load library config (paths, platform, extensions)
3. Count total files (progress bar)
4. For each ROM file:
   a. Extract metadata from filename
   b. Find/create game by title
   c. Calculate hashes (new) or reuse (existing)
   d. Create/update game version
   e. Save metadata file
   f. Update game metadata (dev, pub)
5. Cleanup: Soft-delete games with no files on disk

## 💾 API Response Structure

```json
{
  "id": "game-uuid",
  "title": "Super Mario World",
  "platformId": 2,
  "developer": "Nintendo",
  "versions": [
    {
      "id": "version-uuid",
      "versionName": "PAL",
      "filePath": "/path/to/game.sfc",
      "fileSize": 1048576,
      "md5": "...",
      "sha1": "...",
      "sha256": "..."
    },
    // ... more versions
  ]
}
```

## 📈 Statistics

- **Total files in library dir**: ~13 files
- **Test files**: 11 (mostly unimplemented)
- **Main scanning function**: 207 lines
- **Metadata extraction**: 69 lines (has the bug)
- **Supported platforms**: 15+ (NES, SNES, N64, Genesis, PS1/2, etc.)
- **Hash algorithms**: 3 implemented (MD5, SHA1, SHA256)

## ✅ Verification Points

After reading these documents, you should understand:

1. ✓ How the scanner detects ROM files (extension matching)
2. ✓ How ROM files are identified as game instances (title matching)
3. ✓ Why "Mario (PAL)" and "Mario (NTSC-U)" become 2 games
4. ✓ Where metadata files are created (`{rom}.meta`)
5. ✓ How versions represent game variants
6. ✓ How hashes are calculated and cached
7. ✓ What the data model looks like (Game → Versions)
8. ✓ How to fix the bug (extract parenthetical regions)

## 🎯 Bug Fix Checklist

After understanding the codebase, implement:

1. [ ] Add region pattern extraction for `(REGION)` format
2. [ ] Create separate "cleanedTitle" without regions
3. [ ] Use cleanedTitle in GetOrCreateGame query
4. [ ] Set VersionName from extracted regions
5. [ ] Update jobs.go to pass region to version creation
6. [ ] Write unit tests for title extraction
7. [ ] Test with multiple region files
8. [ ] Verify existing tests still pass
9. [ ] Manual end-to-end test with PAL/NTSC files

## 📚 References

### Critical Code Locations

| What | File | Line(s) |
|------|------|---------|
| Bug happens here | metadata_io.go | 193-229 |
| Bug manifests here | crud.go | 336-338 |
| Version assignment | jobs.go | 211 |
| Main scan loop | jobs.go | 150-245 |
| Hash calculation | hashing.go | 22-52 |

### Key Functions to Modify

```
metadata_io.go:
  ExtractMetadataFromFilename()  ← Add region pattern extraction

jobs.go:
  scanLibrary()                  ← Pass extracted regions to version creation

crud.go:
  GetOrCreateGame()              ← Use cleaned title
  UpsertGameVersion()            ← Use region for VersionName
```

## 🔗 Cross-References Between Documents

- **QUICK_REFERENCE.md**: "Root Cause" → Details in SCANNER_EXPLORATION_SUMMARY.md
- **SCANNER_EXPLORATION_SUMMARY.md**: "Bug Analysis" section → Visual version in SCANNER_FLOW.md
- **SCANNER_ANALYSIS.md**: "Metadata Extraction" → Code in metadata_io.go lines 169-237
- **SCANNER_FLOW.md**: "The Bug" diagram → Bug Description in SCANNER_EXPLORATION_SUMMARY.md

## 📞 Support

If you need to understand a specific aspect:

- **"How are ROMs detected?"** → SCANNER_ANALYSIS.md "ROM File Detection"
- **"Where does the bug happen?"** → QUICK_REFERENCE.md "Root Cause" + SCANNER_FLOW.md
- **"What's the complete process?"** → SCANNER_EXPLORATION_SUMMARY.md "Scanning Process"
- **"How do hashes work?"** → SCANNER_ANALYSIS.md "Hash Calculation & Caching"
- **"What's the schema?"** → SCANNER_EXPLORATION_SUMMARY.md "Complete Data Model"
- **"Where's file X?"** → This document "File Structure Overview"

---

**Created**: 2026-02-20
**Status**: Complete exploration of game library scanner codebase
**Next Step**: Use QUICK_REFERENCE.md to implement the fix
