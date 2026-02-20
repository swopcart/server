# Swopcart Library - Structure Diagrams & Data Flow

## 1. Data Model Relationships

```
┌─────────────────────────────────────────────────────────────────┐
│                        LIBRARY ECOSYSTEM                         │
└─────────────────────────────────────────────────────────────────┘

┌──────────┐
│ Platform │
├──────────┤
│ ID       │ (uint, primary key)
│ Name     │ (unique) "NES", "SNES", etc.
│ Extensions
  JSON     │ [".sfc", ".smc", ".rom"]
└─────┬────┘
      │ 1:N
      │
      ├─────────────────────┐
      │                     │
      v                     v
┌──────────┐          ┌──────────┐
│ Library  │          │  Game    │
├──────────┤          ├──────────┤
│ ID       │          │ ID       │ (uuid)
│ Name     │          │ LibraryID│ (fk)
│ Paths    │          │ PlatformID
│ (JSON    │          │ Title    │ (indexed)
│  array)  │          │ Developer│ (optional)
│ PlatformID          │ Publisher│ (optional)
│ ScanStatus│          │ ReleasedDate
│ LastScanned
  At       │          │ Description
└──────┬───┘          └──────┬───┘
       │ 1:N                │ 1:N
       └────────────────────┤
                            v
                      ┌──────────────┐
                      │ GameVersion  │
                      ├──────────────┤
                      │ ID           │ (uuid)
                      │ GameID       │ (fk)
                      │ FilePath     │ (unique index)
                      │ VersionName  │ "1.0", "PAL", etc.
                      │ FileSize     │
                      │ MD5/SHA1/    │
                      │ SHA256/Blake3│ (hashes)
                      │ MetadataPath │ "file.meta"
                      │ MetadataJSON │ (serialized)
                      └──────────────┘
```

## 2. Scanner Workflow

```
┌─────────────────────────────────────────────────────────────┐
│                   LIBRARY SCAN PROCESS                       │
└─────────────────────────────────────────────────────────────┘

┌─────────────┐
│   START     │ libraryScanJob() triggered
└──────┬──────┘
       │
       v
┌─────────────────────────────────┐
│ VALIDATE SCAN STATE             │
│ - Check CurrentScanJobID        │
│ - Prevent concurrent scans      │
│ - Mark library as "scanning"    │
└──────┬──────────────────────────┘
       │
       v
┌─────────────────────────────────┐
│ FIRST PASS: Count Files         │
│ filepath.Walk(paths)            │
│ Count files matching extensions │
│ progress.UpdateProgress(total,0)│
└──────┬──────────────────────────┘
       │
       v
┌─────────────────────────────────┐
│ SECOND PASS: Process Files      │
│ filepath.Walk(paths) again      │
│                                 │
│ FOR EACH FILE:                  │
│  1. Check extension match       │
│  2. Track in foundFiles map     │
│  3. Get directory name          │
└──────┬──────────────────────────┘
       │
       v
┌─────────────────────────────────────────┐
│ EXTRACT METADATA FROM FILENAME          │
│ "Game (PAL) [vgdb=123].sfc"             │
│                                         │
│ title = "Game"                          │
│ regions = ["PAL"]                       │
│ externalIds = {"vgdb": "123"}           │
│ platform = "SNES"                       │
└──────┬──────────────────────────────────┘
       │
       v
┌─────────────────────────────────────────┐
│ GET OR CREATE GAME                      │
│ Query by: (LibraryID + Title)           │
│ If found: reuse                         │
│ If not: create new                      │
└──────┬──────────────────────────────────┘
       │
       v
┌─────────────────────────────────────────┐
│ HASH CALCULATION (First time only)      │
│ Check if version exists by FilePath     │
│ If new: Calculate MD5/SHA1/SHA256       │
│ If exists: Reuse cached hashes          │
│ If error: Use empty hashes              │
└──────┬──────────────────────────────────┘
       │
       v
┌──────────────────────────────────────────┐
│ UPSERT GAME VERSION                      │
│ Create/Update in GameVersion table       │
│ Set: FilePath (unique index)             │
│      VersionName (from dirname)          │
│      FileSize, Hashes                    │
│      MetadataPath                        │
└──────┬─────────────────────────────────┘
       │
       v
┌──────────────────────────────────────────┐
│ SAVE METADATA TO FILE                    │
│ Check if {filePath}.meta exists          │
│ If not: Write TOML file with metadata    │
│ Format: kebab-case                       │
└──────┬─────────────────────────────────┘
       │
       v
┌──────────────────────────────────────────┐
│ UPDATE GAME METADATA (if extracted)      │
│ If developer/publisher found:            │
│   Update Game.developer/publisher        │
└──────┬─────────────────────────────────┘
       │
       v
┌──────────────────────────────────────────┐
│ UPDATE PROGRESS                          │
│ progress.UpdateProgress(total, processed)│
└──────┬─────────────────────────────────┘
       │
       v
┌──────────────────────────────────────────┐
│ CLEANUP: Soft-delete missing games       │
│ For each game:                           │
│   Check if any version still on disk     │
│   If not: soft-delete game               │
└──────┬─────────────────────────────────┘
       │
       v
┌──────────────────────────────────────────┐
│ MARK SCAN COMPLETE                       │
│ Set library.ScanStatus = "idle"          │
│ Set library.LastScannedAt = now()        │
│ Clear CurrentScanJobID                   │
└──────┬─────────────────────────────────┘
       │
       v
┌──────────────┐
│   SUCCESS    │
└──────────────┘
```

## 3. Metadata Format Transformation

```
┌──────────────────────────────────────────────────────────┐
│           METADATA TRANSFORMATION PIPELINE               │
└──────────────────────────────────────────────────────────┘

FILESYSTEM (TOML - kebab-case)
════════════════════════════════════════════════════════
zelda.nes.meta:
  title = "The Legend of Zelda"
  platform = "NES"
  release-date = "1986-07-27"
  developer = "Nintendo"
  publisher = "Nintendo"

  [external-ids]
  vgdb = "12345"

  regions = ["NTSC", "PAL"]
  tags = ["action"]

  [[versions]]
  name = "1.0"
  filename = "zelda.nes"
════════════════════════════════════════════════════════
              │
              │ LoadMetadataFromFile()
              │
              v
GO STRUCT (GameMetadata)
════════════════════════════════════════════════════════
{
  Title: "The Legend of Zelda",
  Platform: "NES",
  ReleaseDate: "1986-07-27",
  Developer: "Nintendo",
  Publisher: "Nintendo",
  ExternalIDs: map[string]string{
    "vgdb": "12345",
  },
  Regions: []string{"NTSC", "PAL"},
  Tags: []string{"action"},
  Versions: []VersionMetadata{
    {
      Name: "1.0",
      Filename: "zelda.nes",
    },
  },
}
════════════════════════════════════════════════════════
              │
              │ MetadataToJSON()
              │ (rename fields: release-date → releaseDate)
              │
              v
JSON STRUCT (GameMetadataJSON)
════════════════════════════════════════════════════════
{
  "title": "The Legend of Zelda",
  "platform": "NES",
  "releaseDate": "1986-07-27",
  "developer": "Nintendo",
  "publisher": "Nintendo",
  "externalIds": {
    "vgdb": "12345"
  },
  "regions": ["NTSC", "PAL"],
  "tags": ["action"],
  "versions": [
    {
      "name": "1.0",
      "filename": "zelda.nes"
    }
  ]
}
════════════════════════════════════════════════════════
              │
              │ MetadataToString()
              │ json.Marshal()
              │
              v
DATABASE (TEXT field - JSON string)
════════════════════════════════════════════════════════
GameVersion.MetadataJSON:
'{"title":"The Legend of Zelda","platform":"NES",...}'
════════════════════════════════════════════════════════
              │
              │ API Response (json.Marshal)
              │
              v
API RESPONSE (JSON)
════════════════════════════════════════════════════════
HTTP/1.1 200 OK
Content-Type: application/json

{
  "title": "The Legend of Zelda",
  "platform": "NES",
  "releaseDate": "1986-07-27",
  "developer": "Nintendo",
  "publisher": "Nintendo",
  "externalIds": {"vgdb": "12345"},
  "regions": ["NTSC", "PAL"],
  "tags": ["action"],
  "versions": [...]
}
════════════════════════════════════════════════════════
```

## 4. Filename Parsing Rules Diagram

```
┌──────────────────────────────────────────────────────────┐
│           FILENAME EXTRACTION PIPELINE                    │
└──────────────────────────────────────────────────────────┘

INPUT:  "Super Mario World (PAL) [vgdb=12345].sfc"
        └─ Full filename with extension

        │
        v
STEP 1: Remove Extension
        "Super Mario World (PAL) [vgdb=12345]"

        │
        v
STEP 2: Extract vgdb ID [vgdb=XXXXX]
        ExternalIDs["vgdb"] = "12345"
        Remaining: "Super Mario World (PAL) "

        │
        v
STEP 3: Extract Region (.pal, .ntsc, .ntsc-u, .ntsc-j)
        Regions = ["PAL"]
        Remaining: "Super Mario World "

        │
        v
STEP 4: Extract Region [USA], [EUR], [JPN], [AUS], [CAN]
        (No matches)
        Remaining: "Super Mario World "

        │
        v
STEP 5: Extract Version/Tag (vX.X, Rev X, Demo, Beta)
        (v1.0) pattern removed from title
        Tags populated if "Demo" or "Beta"
        Remaining: "Super Mario World "

        │
        v
STEP 6: Clean Spaces
        Collapse whitespace
        Trim edges
        Title = "Super Mario World"

        │
        v
OUTPUT:
{
  Title: "Super Mario World",
  Platform: "SNES",
  Regions: ["PAL"],
  ExternalIDs: {"vgdb": "12345"},
  Tags: []
}
```

## 5. File Organization Patterns

```
┌──────────────────────────────────────────────────────────┐
│              FLAT FILE STRUCTURE                          │
└──────────────────────────────────────────────────────────┘

/mnt/games/nes/
├── zelda.nes
├── zelda.nes.meta          ← Metadata for zelda.nes
├── mario.nes
├── mario.nes.meta          ← Metadata for mario.nes
└── donkey_kong.nes

Scanner behavior:
  File 1: zelda.nes
    └─ Game: {Title: "Zelda"}
       └─ Version: {FilePath: /mnt/games/nes/zelda.nes}
            └─ Metadata: /mnt/games/nes/zelda.nes.meta

  File 2: mario.nes
    └─ Game: {Title: "Mario"}
       └─ Version: {FilePath: /mnt/games/nes/mario.nes}
            └─ Metadata: /mnt/games/nes/mario.nes.meta


┌──────────────────────────────────────────────────────────┐
│           PER-GAME DIRECTORY STRUCTURE                    │
└──────────────────────────────────────────────────────────┘

/mnt/games/nes/
├── zelda/
│   ├── zelda_1.0.nes
│   ├── zelda_1.0.nes.meta  ← Current: per-file metadata
│   ├── zelda_pal.nes
│   └── zelda_pal.nes.meta  ← Each file gets its own metadata
│
├── mario/
│   ├── mario.nes
│   └── mario.nes.meta
│
└── donkey_kong.nes
    └── donkey_kong.nes.meta

Scanner behavior (current):
  File 1: zelda/zelda_1.0.nes
    └─ Game: {Title: "Zelda_1"}  ← Extracted from filename
       └─ Version: {FilePath: /mnt/games/nes/zelda/zelda_1.0.nes}
            └─ Metadata: /mnt/games/nes/zelda/zelda_1.0.nes.meta

  File 2: zelda/zelda_pal.nes
    └─ Game: {Title: "Zelda"}  ← DIFFERENT! (region stripped)
       └─ Version: {FilePath: /mnt/games/nes/zelda/zelda_pal.nes}
            └─ Metadata: /mnt/games/nes/zelda/zelda_pal.nes.meta

PROBLEM: zelda_1.0 and zelda_pal create SEPARATE games!


┌──────────────────────────────────────────────────────────┐
│         INTENDED: SHARED METADATA PER DIRECTORY           │
└──────────────────────────────────────────────────────────┘

/mnt/games/nes/
├── zelda/
│   ├── meta.toml           ← SHARED metadata for all versions
│   │   Contains: [versions] entries for all files
│   ├── zelda_1.0.nes
│   ├── zelda_pal.nes
│   └── zelda_enhanced.nes
│
├── mario/
│   ├── meta.toml
│   ├── mario.nes
│   ├── mario_pal.nes
│   └── mario_enhanced.nes
│
└── flat_game.nes           ← Flat files still supported
    └── flat_game.nes.meta

Intended behavior:
  Directory zelda/
    └─ Game: {Title: "Zelda"}
       ├─ Version: {VersionName: "1.0", FilePath: zelda_1.0.nes}
       ├─ Version: {VersionName: "PAL", FilePath: zelda_pal.nes}
       └─ Version: {VersionName: "Enhanced", FilePath: zelda_enhanced.nes}
       └─ Single metadata file shared by all versions

  File flat_game.nes
    └─ Game: {Title: "Flat Game"}
       └─ Version: {FilePath: flat_game.nes}
            └─ Metadata: flat_game.nes.meta
```

## 6. Version Identification

```
┌──────────────────────────────────────────────────────────┐
│           HOW VERSIONS ARE NAMED                          │
└──────────────────────────────────────────────────────────┘

Input: /mnt/games/snes/Super Mario World/mario_v1.0.sfc

1. Extract directory name
   dirName = filepath.Base(filepath.Dir(filePath))
   dirName = "Super Mario World"

2. Version name assignment
   GameVersion.VersionName = dirName = "Super Mario World"

   Problem: All files in same directory get same VersionName!

   Expected: Extract from filename
   - mario_v1.0.sfc → VersionName: "1.0"
   - mario_pal.sfc → VersionName: "PAL"
   - mario_enhanced.sfc → VersionName: "Enhanced"

Current implementation (jobs.go line 211):
  svc.UpsertGameVersion(ctx, game.ID, filePath, dirName, ...)
  //                                              ^^^^^
  //                                   Uses directory name


Version name sources (priority order):
1. TOML metadata: Versions[].name
2. Filename pattern: (v1.0), (PAL), (Demo)
3. Region tags: .pal, .ntsc-u, .ntsc-j
4. Directory name (current fallback)
```

## 7. Hash Calculation & Caching

```
┌──────────────────────────────────────────────────────────┐
│          HASH CACHING STRATEGY                            │
└──────────────────────────────────────────────────────────┘

Scan Start
│
v
For each ROM file
│
├─ Check if GameVersion exists by FilePath
│
├─ If NOT exists (new file):
│  │
│  ├─ CalculateFileHashes()
│  │  ├─ Read file in 64KB chunks
│  │  ├─ Calculate MD5 simultaneously
│  │  ├─ Calculate SHA1 simultaneously
│  │  ├─ Calculate SHA256 simultaneously
│  │  └─ Store all hashes
│  │
│  └─ Store hashes in GameVersion
│
├─ If exists (file already scanned):
│  │
│  ├─ Retrieve hashes from DB
│  └─ Reuse without recalculation
│
└─ If error accessing file:
   └─ Use empty hashes (no error thrown)

Benefits:
- Avoid expensive hash recalculation on each scan
- Memory efficient (streaming, not loading whole file)
- Supports large ROM files (1GB+)

Performance:
- First scan: CalculateFileHashes() time
- Subsequent scans: DB lookup only (fast)
```

## 8. Library Lifecycle

```
┌──────────────────────────────────────────────────────────┐
│           LIBRARY LIFECYCLE DIAGRAM                       │
└──────────────────────────────────────────────────────────┘

1. CREATE LIBRARY
   └─ Admin: POST /api/v0/libraries
      {
        "name": "My NES Collection",
        "description": "NES games",
        "platformId": 1,
        "paths": ["/mnt/games/nes", "/mnt/roms/nes"]
      }

   └─ Service validation
      ├─ Verify platform exists
      ├─ Validate all paths exist and are directories
      └─ Create Library record

   └─ AUTO-TRIGGER SCAN
      └─ Enqueue "library.scan" job
         └─ JobExecution starts
            └─ scanLibrary() runs


2. IDLE STATE
   └─ Library.ScanStatus = "idle"
   └─ Next scan: scheduled (0 0 3 * * *) or manual


3. MANUAL SCAN
   └─ Admin: POST /api/v0/libraries/{id}/scan
      └─ Enqueue "library.scan" job
         └─ JobExecution starts


4. SCHEDULED SCAN
   └─ Cron trigger at 3 AM daily
      └─ libraryScanJob() runs for all libraries
         └─ scanLibrary() for each


5. SCAN IN PROGRESS
   └─ Library.ScanStatus = "scanning"
   └─ Library.CurrentScanJobID = {uuid}
   └─ Prevents concurrent scans


6. UPDATE GAMES/METADATA
   └─ Admin: PATCH /api/v0/games/{gameId}
      {
        "title": "New Title",
        "developer": "New Dev"
      }
   └─ Updates Game record
   └─ Updates .meta files
   └─ Updates GameVersion.MetadataJSON


7. DELETE LIBRARY
   └─ Admin: DELETE /api/v0/libraries/{id}
      └─ Soft-delete Library
         └─ Games/GameVersions not deleted (just disconnected)
            (Could add cascade delete if needed)


8. SCAN DETECTS MISSING FILE
   └─ File deleted from disk
   └─ Scanner doesn't find it in foundFiles map
   └─ soft-delete Game record
```
