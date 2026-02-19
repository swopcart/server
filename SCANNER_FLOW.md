# Game Library Scanner - Data Flow Diagram

## Complete Scanning Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      LIBRARY SCAN JOB TRIGGERED                             │
│        (Manual via API or Scheduled at 3 AM via cron: "0 0 3 * * *")        │
└────────────────────────────┬────────────────────────────────────────────────┘
                             │
                             ▼
        ┌────────────────────────────────────┐
        │   libraryScanJob() [jobs.go:37]    │
        │   - Get list of libraries to scan  │
        │   - Call scanLibrary() for each    │
        └────────────┬───────────────────────┘
                     │
                     ▼
        ┌────────────────────────────────────┐
        │ scanLibrary() [jobs.go:69]         │
        │ Main scanning orchestrator         │
        └────────────┬───────────────────────┘
                     │
            ┌────────┴────────┐
            │                 │
            ▼                 ▼
    ┌──────────────────┐ ┌──────────────────┐
    │  FIRST PASS      │ │  SECOND PASS     │
    │  Count files     │ │  Process files   │
    │  [lines 126-144] │ │  [lines 148-245] │
    └──────────────────┘ └────────┬─────────┘
                                  │
                  ┌───────────────┴───────────────┐
                  │ For each matched ROM file:    │
                  └───────────────┬───────────────┘
                                  │
                     ┌────────────┴────────────┐
                     │                         │
                     ▼                         ▼
        ┌─────────────────────────┐  ┌─────────────────────────┐
        │ EXTRACT METADATA        │  │ MATCH GAME              │
        │ from filename           │  │ [crud.go:331]           │
        │ [metadata_io.go:169]    │  │ GetOrCreateGame()       │
        │                         │  │                         │
        │ Input:                  │  │ Query by:               │
        │ "Super Mario World      │  │ - LibraryID +           │
        │  (PAL) [vgdb=123].sfc"  │  │ - Title                 │
        │                         │  │                         │
        │ Extracts:              │  │ Current BUG:            │
        │ - title: "Super Mario  │  │ Title INCLUDES region   │
        │   World (PAL)"         │  │ so PAL and NTSC-U are   │
        │ - regions: ["PAL"]     │  │ different games!        │
        │ - vgdb: "123"          │  │                         │
        └─────────────┬───────────┘  └────────────┬────────────┘
                      │                           │
                      │                      If NOT found:
                      │         ┌──────────────────────────┐
                      │         │ CREATE NEW GAME          │
                      │         │ [crud.go:355]            │
                      │         │ Game{                    │
                      │         │   ID: uuid.New(),        │
                      │         │   Title: full title,     │
                      │         │   LibraryID,             │
                      │         │   PlatformID             │
                      │         │ }                        │
                      │         └────────┬─────────────────┘
                      │                  │
                      ▼                  ▼
        ┌──────────────────────────────────────┐
        │ CALCULATE FILE HASHES                │
        │ [hashing.go:22]                      │
        │                                      │
        │ For NEW files:                       │
        │ - MD5 (32 hex)                       │
        │ - SHA1 (40 hex)                      │
        │ - SHA256 (64 hex)                    │
        │ - Blake3 (reserved)                  │
        │                                      │
        │ Streaming calculation to save RAM    │
        │                                      │
        │ For EXISTING files (by FilePath):    │
        │ - Reuse cached hashes from DB        │
        └────────────────┬─────────────────────┘
                         │
                         ▼
        ┌──────────────────────────────────────┐
        │ CREATE/UPDATE GAME VERSION           │
        │ [crud.go:371]                        │
        │ UpsertGameVersion()                  │
        │                                      │
        │ Creates/Updates by FilePath uniqueness
        │                                      │
        │ GameVersion{                         │
        │   GameID: (from above)               │
        │   FilePath: "/path/game.sfc"         │
        │   VersionName: dirName               │
        │   FileSize: info.Size()              │
        │   MD5/SHA1/SHA256/Blake3: hashes     │
        │ }                                    │
        │                                      │
        │ BUG: VersionName uses dirName,       │
        │ should use extracted region          │
        └────────────────┬─────────────────────┘
                         │
                         ▼
        ┌──────────────────────────────────────┐
        │ SAVE METADATA FILE TO DISK           │
        │ [jobs.go:216-222]                    │
        │                                      │
        │ Metadata File Path:                  │
        │ {filePath}.meta                      │
        │ e.g., "game.sfc.meta"                │
        │                                      │
        │ Only created if NOT EXISTS           │
        │ User edits are preserved             │
        │                                      │
        │ Format: TOML (kebab-case)            │
        │ title = "..."                        │
        │ regions = ["PAL"]                    │
        │ [external-ids]                       │
        │ vgdb = "123"                         │
        └────────────────┬─────────────────────┘
                         │
                         ▼
        ┌──────────────────────────────────────┐
        │ UPDATE GAME METADATA                 │
        │ [jobs.go:225-236]                    │
        │                                      │
        │ If extracted metadata has:           │
        │ - developer                          │
        │ - publisher                          │
        │ Updates Game record with these       │
        │ (from database.Game fields)          │
        └────────────────┬─────────────────────┘
                         │
                         ▼
        ┌──────────────────────────────────────┐
        │ UPDATE PROGRESS                      │
        │ [jobs.go:239]                        │
        │                                      │
        │ progress.UpdateProgress(             │
        │   totalRecords,                      │
        │   processedRecords,                  │
        │   filename                           │
        │ )                                    │
        └────────────────┬─────────────────────┘
                         │
                    ┌────┴────┐
                    │          │
                    ▼          ▼
            ┌──────────────────────────┐
            │ More files?              │
            │ Continue loop [line 150] │
            │ OR Done?                 │
            └──────────────────────────┘
                    │
                    ├─ YES (more files): Go back to EXTRACT METADATA
                    │
                    └─ NO (done): Continue to cleanup
                                  │
                                  ▼
        ┌──────────────────────────────────────┐
        │ SOFT-DELETE MISSING GAMES            │
        │ [jobs.go:247-271]                    │
        │                                      │
        │ For each game in library:            │
        │ - Check if ANY version still exists  │
        │   on disk (check foundFiles map)     │
        │ - If no versions exist: soft-delete  │
        │   the Game record                    │
        └──────────────────────────────────────┘
```

## The Bug: Title Extraction Issue

```
BEFORE SCAN (Disk):
├── Super Mario World (PAL).sfc
└── Super Mario World (NTSC-U).sfc

FILENAME PARSING:
┌──────────────────────────────┐  ┌──────────────────────────┐
│ "Super Mario World (PAL)"    │  │ "Super Mario (NTSC-U)"   │
│ .sfc                         │  │ .sfc                     │
│                              │  │                          │
│ ExtractMetadataFromFilename: │  │ ExtractMetadataFromFilename:
│                              │  │                          │
│ Result:                      │  │ Result:                  │
│ {                            │  │ {                        │
│   Title: "Super Mario World  │  │   Title: "Super Mario    │
│   (PAL)"  ← BUG! Region      │  │   World (NTSC-U)"        │
│           included in title  │  │   ← BUG! Different!      │
│   Regions: ["PAL"]           │  │   Regions: ["NTSC"]      │
│ }                            │  │ }                        │
└────────────┬──────────────────┘  └──────────────┬───────────┘
             │                                    │
             ▼                                    ▼
┌──────────────────────────────┐  ┌──────────────────────────┐
│ GetOrCreateGame()            │  │ GetOrCreateGame()        │
│ Query:                       │  │ Query:                   │
│ WHERE libraryID = X          │  │ WHERE libraryID = X      │
│   AND title = "Super Mario   │  │   AND title = "Super     │
│   World (PAL)"               │  │   Mario World (NTSC-U)"  │
│                              │  │                          │
│ Result: NO MATCH             │  │ Result: NO MATCH         │
│ Creates new Game #1          │  │ Creates new Game #2      │
└──────────────────────────────┘  └──────────────────────────┘
             │                                    │
             ▼                                    ▼
        ┌────────────┐                      ┌────────────┐
        │  Game #1   │                      │  Game #2   │
        │ Title: "SM │                      │ Title: "SM │
        │ World PAL" │                      │ World NTSC │
        └────────────┘                      └────────────┘
             │                                    │
             ▼                                    ▼
    ┌──────────────────┐                ┌──────────────────┐
    │  GameVersion #1  │                │  GameVersion #2  │
    │  FilePath: .../  │                │  FilePath: .../  │
    │  (PAL).sfc       │                │  (NTSC-U).sfc    │
    │  VersionName:    │                │  VersionName:    │
    │  (dirname)       │                │  (dirname)       │
    └──────────────────┘                └──────────────────┘

AFTER SCAN (Database):
❌ TWO GAMES CREATED (BUG!)
├── Game: "Super Mario World (PAL)"
│   └── Version: (PAL).sfc
└── Game: "Super Mario World (NTSC-U)"
    └── Version: (NTSC-U).sfc

EXPECTED RESULT:
✓ ONE GAME (CORRECT)
└── Game: "Super Mario World"
    ├── Version (PAL): (PAL).sfc
    └── Version (NTSC-U): (NTSC-U).sfc
```

## Data Flow: From API Response to Frontend

```
Database State:
┌─────────────────────────────────────┐
│ Game: Super Mario World             │
│ ID: 550e8400-e29b-41d4-a716-...     │
│ Title: "Super Mario World (PAL)"    │  ← BUG: Title has region
│ Developer: "Nintendo"               │
│                                     │
│ Versions:                           │
│ ├─ GameVersion                      │
│ │  ID: 550e8400-e29b-41d4-a716-aaa  │
│ │  VersionName: ".../game.sfc"      │  ← Directory name!
│ │  FilePath: ".../Super Mario...    │
│ │           (PAL).sfc"              │
│ │  MD5: "abc123..."                 │
│ │                                   │
│ └─ GameVersion (NOT CREATED, BUG!)  │
│    For NTSC-U version               │
└─────────────────────────────────────┘
         │
         │ GetGameByID()
         │ [queries.go:80]
         │
         ▼
┌─────────────────────────────────────┐
│ GameWithVersions Response           │
│ {                                   │
│   "id": "550e8400...",              │
│   "title": "Super Mario World (PAL)",
│   "platformId": 2,                  │
│   "developer": "Nintendo",          │
│   "versions": [                     │
│     {                               │
│       "id": "550e8400...",          │
│       "versionName": ".../game.sfc",│
│       "filePath": ".../game.sfc"    │
│     }                               │
│   ]                                 │
│ }                                   │
└─────────────────────────────────────┘
         │
         │ HTTP 200
         │
         ▼
┌─────────────────────────────────────┐
│ Frontend receives WRONG structure:   │
│ - User sees 2 separate games        │
│ - Each with 1 version               │
│ - Should see 1 game with 2 versions │
└─────────────────────────────────────┘
```

## Metadata File Example

```
Disk Layout:
library/
├── Super Mario World (PAL).sfc
├── Super Mario World (PAL).sfc.meta  ← Created by scanner
└── Super Mario World (NTSC-U).sfc
    (NOTE: .meta file should be created for this too!)

File: Super Mario World (PAL).sfc.meta
─────────────────────────────────────
title = "Super Mario World"
platform = "SNES"
developer = "Nintendo"
publisher = "Nintendo"
release-date = "1990-11-21"
regions = ["PAL"]
tags = []

[external-ids]
vgdb = "123456"

[[versions]]
name = "PAL"
filename = "Super Mario World (PAL).sfc"
regions = ["PAL"]
```
