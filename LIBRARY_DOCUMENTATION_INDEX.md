# Swopcart Library Documentation Index

## Quick Navigation

This documentation explores the swopcart game library scanning system in depth. Start with the overview and drill down as needed.

### 📄 Documentation Files

1. **[LIBRARY_DOCUMENTATION_INDEX.md](LIBRARY_DOCUMENTATION_INDEX.md)** (this file)
   - Navigation guide for all library documentation
   - Quick reference to key files in codebase

2. **[LIBRARY_METADATA_EXPLORATION.md](LIBRARY_METADATA_EXPLORATION.md)** (660 lines)
   - Complete analysis of library structure expectations
   - Metadata file format (TOML schema + examples)
   - Database models (Game, GameVersion)
   - Metadata I/O operations
   - Filename parsing rules
   - Scanner workflow
   - Real-world examples

3. **[LIBRARY_STRUCTURE_DIAGRAMS.md](LIBRARY_STRUCTURE_DIAGRAMS.md)** (587 lines)
   - Visual ASCII diagrams of key concepts
   - Data model relationships
   - Scanner workflow diagram
   - Metadata transformation pipeline
   - Filename parsing pipeline
   - File organization patterns
   - Hash calculation & caching
   - Library lifecycle

4. **[LIBRARY_KEY_FINDINGS.md](LIBRARY_KEY_FINDINGS.md)** (452 lines)
   - Critical discoveries and insights
   - Two metadata systems explanation
   - Directory traversal approach (two-pass)
   - **Game matching logic (root cause of issues)**
   - Metadata file naming inconsistencies
   - Version naming problems
   - Critical limitations summary
   - Design principles observed
   - Enhancement recommendations

---

## 🎯 Key Discoveries

### Two Metadata Systems
- **Legacy** (`metadata.go`): Has UUID, keyed versions, runner support - **NOT USED**
- **Active** (`metadata_io.go`): No UUID, array versions, simple structure - **ACTUALLY USED**

### Critical Issues Found
1. **Game Matching Bug**: Games matched by title only - causes version splitting!
   - Example: `zelda_1.0.nes` and `zelda_pal.nes` create TWO games

2. **Version Naming**: All files in a directory get same version name (dirname)
   - Should extract from filename pattern: `(v1.0)`, `(PAL)`, etc.

3. **Metadata Handling**: Per-file metadata, not directory-level
   - Plan shows directory-level `meta.toml`, implementation does per-file

---

## 🗂️ Codebase Structure

### Core Files
- **`internal/database/models.go`** (Game, GameVersion, Platform, Library models)
- **`internal/services/library/metadata.go`** (legacy Metadata struct - not used)
- **`internal/services/library/metadata_io.go`** (GameMetadata, TOML/JSON conversion)
- **`internal/services/library/service.go`** (platform loading, initialization)
- **`internal/services/library/crud.go`** (game/version creation/update logic)
- **`internal/services/library/jobs.go`** (scanner workflow - TWO-PASS algorithm)
- **`internal/services/library/hashing.go`** (streaming file hashing)
- **`internal/services/library/queries.go`** (game lookup queries)

### Test Files
- **`internal/services/library/metadata_test.go`** (legacy Metadata TOML/JSON round-trip)
- **`internal/services/library/metadata_io_test.go`** (GameMetadata load/save/parse)
- **`internal/services/library/crud_test.go`** (game/version CRUD operations)
- **`internal/services/library/scan_test.go`** (stub tests - not implemented)

### API Handlers
- **`internal/www/api/v0/libraries.go`** (library endpoints)
- **`internal/www/api/v0/games.go`** (game endpoints)

---

## 📋 Quick Reference

### Metadata File Format

**On Disk (TOML - kebab-case):**
```toml
title = "Game Title"
platform = "NES"
developer = "Developer"
release-date = "1990-01-01"

[external-ids]
vgdb = "12345"

regions = ["NTSC"]
tags = ["action"]

[[versions]]
name = "1.0"
filename = "game_1.0.nes"
```

**In API (JSON - camelCase):**
```json
{
  "title": "Game Title",
  "platform": "NES",
  "developer": "Developer",
  "releaseDate": "1990-01-01",
  "externalIds": {"vgdb": "12345"},
  "regions": ["NTSC"],
  "tags": ["action"],
  "versions": [{"name": "1.0", "filename": "game_1.0.nes"}]
}
```

### Directory Structure

**Flat Files:**
```
/games/nes/
  ├── game1.nes
  ├── game1.nes.meta
  ├── game2.nes
  └── game2.nes.meta
```

**Per-Game Directory (Intended but Partially Implemented):**
```
/games/nes/
  ├── game1/
  │   ├── game1_1.0.nes
  │   ├── game1_pal.nes
  │   └── meta.toml  (intended: shared by all files)
  └── game2/
      ├── game2.nes
      └── meta.toml
```

---

## 🔍 Detailed Analysis Map

### For Understanding Metadata
→ See **LIBRARY_METADATA_EXPLORATION.md**
- Section 2: Metadata File Format (complete schema)
- Section 5: Metadata I/O Operations
- Section 6: Filename Parsing Rules

### For Understanding Data Models
→ See **LIBRARY_METADATA_EXPLORATION.md**
- Section 3: Database Models for Game & GameVersion
- **LIBRARY_STRUCTURE_DIAGRAMS.md** Section 1: Data Model Relationships

### For Understanding Scanner
→ See **LIBRARY_METADATA_EXPLORATION.md**
- Section 7: Scanner Flow
- **LIBRARY_STRUCTURE_DIAGRAMS.md** Section 2: Scanner Workflow

### For Understanding Issues
→ See **LIBRARY_KEY_FINDINGS.md**
- Section: Game Matching Logic - The Root Cause
- Section: Version Naming - Another Issue
- Section: Critical Limitations Summary

### For Visual Understanding
→ See **LIBRARY_STRUCTURE_DIAGRAMS.md**
- Section 1: Data Model Relationships (ER diagram)
- Section 2: Scanner Workflow (flowchart)
- Section 3: Metadata Transformation (pipeline)
- Section 5: File Organization Patterns

---

## 📊 Statistics

| Aspect | Current | Documented |
|--------|---------|-----------|
| Database Models | 4 (Platform, Library, Game, GameVersion) | ✓ |
| Metadata Struct Types | 2 (Metadata, GameMetadata) | ✓ |
| Filename Parsing Patterns | 8+ patterns | ✓ |
| File Organization Patterns | 2 (Flat, Per-game Dir) | ✓ |
| Scanner Passes | 2 (Count, Process) | ✓ |
| Hash Algorithms | 4 (MD5, SHA1, SHA256, Blake3) | ✓ |
| Default Platforms | 15 | ✓ |
| Critical Issues Found | 3 major | ✓ |
| Enhancement Recommendations | 10 | ✓ |

---

## 🚀 Next Steps for Implementation

### High Priority
1. Review **LIBRARY_KEY_FINDINGS.md** "Game Matching Logic" section
2. Understand the bug causing version splitting
3. Plan fix for title extraction to preserve region/version info

### Medium Priority
4. Review **LIBRARY_METADATA_EXPLORATION.md** "Current vs Intended Directory Structure"
5. Implement directory-level metadata support
6. Extract version names from filenames

### Low Priority
7. Unify metadata systems (merge legacy Metadata into GameMetadata)
8. Optimize scanner to single-pass
9. Add support for modern platforms

---

## 📚 Related Documents in Repository

- **[GAME_LIBRARY_PLAN.md](GAME_LIBRARY_PLAN.md)** - Original design and implementation plan
- **[SCANNER_ANALYSIS.md](SCANNER_ANALYSIS.md)** - Previous scanner analysis
- **[SCANNER_FLOW.md](SCANNER_FLOW.md)** - Scanner flow details
- **[CLAUDE.md](CLAUDE.md)** - Project guidelines and architecture

---

## 🤝 Document Generation

These documents were generated by analyzing:
- Source code in `internal/services/library/`
- Database models in `internal/database/models.go`
- API handlers in `internal/www/api/v0/`
- Test files in `internal/services/library/`
- Planning documents (GAME_LIBRARY_PLAN.md, etc.)

**Generated**: 2026-02-20
**Status**: Complete exploration phase
**Files**: 3 comprehensive markdown documents (1699 lines)

---

## Questions & Clarifications

### Should we use the legacy Metadata struct?
The `metadata.go` Metadata struct has:
- UUID field (instance identity)
- Keyed versions (map[string]Version)
- RunnerMeta per version
- Per-version hashes

The `metadata_io.go` GameMetadata has:
- No UUID (generic metadata)
- Array versions
- Simple regions/filename tracking

**Decision needed**: Merge these or keep separate?

### How should directory-level metadata work?
Current implementation: `game.nes.meta` (per-file)
Planned implementation: `game_directory/meta.toml` (per-directory)

**Decision needed**: Support both? Migrate to directory-based?

### How to fix game title extraction?
Current problem: `zelda_1.0.nes` → title "zelda_1", `zelda_pal.nes` → title "zelda"
Creates two games instead of two versions of one game.

**Solution options**:
1. Enhanced filename parsing (extract version separately)
2. Match by directory + extracted base title
3. Require manual metadata file for version grouping
