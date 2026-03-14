# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**swopcart** is a self-hosted video game library manager (similar to Jellyfin, but for games). The name refers to "swapping cartridges" (inspired by Stop 'n' Swop from Banjo-Kazooie).

### Vision
- Point at directories per platform (supports diverse platforms: retro consoles, PC, DOS, modern systems, etc.)
- Automatically identify games and make them downloadable to clients
- Sync save files across devices (PC, phone, Steam Deck, etc.)
- Multi-user support for households
- Run acceptably on low-power hardware (ARM64 NAS with 1GB RAM)

### Current Status
Game library feature complete: platforms, libraries, scanning, metadata, downloads, and frontend browser UI all implemented.

## Active Development

### Recently Completed (2026-02-20)
- **Game library service**: ✓ Complete
  - Platform config (embedded `platforms.toml`, auto-seeded on first run)
  - Library management (create/update/delete with multi-path support)
  - Directory-based and flat-file ROM scanner with sidecar metadata
  - External ID markers in filenames (e.g. `[igdb=123]`)
  - Streaming hash calculation (MD5, SHA1, SHA256) via `io.MultiWriter`
  - Concurrent scan prevention via `Library.CurrentScanJobID`
  - Library reimport job (hard-delete + rescan)
  - Full game browser frontend with search, filters, and download

### Next Steps
- ROM identification via hash lookups (IGDB or similar)
- Save file sync system with conflict resolution
- User-specific game libraries / access control

### Design Decisions
- **Performance target**: Must run on ARM64 NAS with 1GB RAM
- **Single binary deployment**: Frontend embedded via go:embed for easy self-hosting
- **Stateless auth**: JWT sessions to minimize memory footprint
- **Streaming-first**: Download/serve files via streaming to avoid loading in memory

## Build & Development Commands

### Backend (Go)
```bash
go build -v ./...                                    # Build all packages
go test -v -race ./...                               # Run all tests
go test -v ./internal/services/session              # Test specific package
go test -v -run TestCreateSession ./internal/services/session  # Run single test
golangci-lint run --timeout=5m                       # Lint
go generate -v ./...                                 # Rebuild embedded frontend assets
```

### Frontend (in frontend/ directory)
```bash
npm run dev          # Dev server (proxies /api to localhost:8000)
npm run build        # Production build
npm run lint         # ESLint
npm run format       # Prettier write
npm run format:check # Prettier check
```

### Full Build
```bash
cd frontend && npm ci && npm run build  # Build frontend
go generate -v ./...                    # Embed frontend in Go binary
go build -v ./cmd/swopcartd             # Build server
```

### Local Development
```bash
go run ./cmd/swopcartd/main.go  # Terminal 1: Backend (needs PostgreSQL + config.toml)
cd frontend && npm run dev       # Terminal 2: Frontend with hot reload

# Or use Justfile for common tasks:
just serve      # Run both backend and frontend in parallel
just test       # Run all Go tests
just lint       # Lint Go and TypeScript
just fmt        # Format Go and TypeScript
just local-db   # Start local PostgreSQL via Docker Compose
```

### Testing
Tests require PostgreSQL. Configure via `SWOPCART_TEST_DB` env var or use default:
```bash
# Default: postgres://swopcart:swopcart@localhost:5432/swopcart_test
SWOPCART_TEST_DB="postgres://user:pass@host:5432/dbname" go test -v -race ./...
```

### Testing Philosophy
This project prioritizes **behavioral testing over implementation testing**. Tests should verify what a system does, not how it does it internally.

**Core principles:**
- **External tests preferred**: Use `package X_test` instead of `package X` to test only the public API
- **Behavior, not internals**: Test observable behavior (API responses, database state, execution results) rather than internal fields, private functions, or implementation details
- **Use testkit**: All service tests use `testkit.New(t)` for isolation with automatic DB transaction rollback
- **Async tests**: Use `testkit.WithoutTransaction()` for tests with background workers/goroutines; call `tk.ResetDB()` before and after test

**What to test:**
- Public API behavior (RegisterHandler, EnqueueJob, GetExecution)
- Observable outcomes (job executes, database records created/updated, errors returned)
- Integration behavior (parameters merge correctly, queues process independently)
- Edge cases (missing handlers, invalid schedules, concurrent execution)

**What NOT to test:**
- Private fields (svc.handlers, svc.cron, svc.workers)
- Private functions (serializeParams, mergeParams, getHostname)
- Implementation details (checking internal state, testing helpers)

**Example:**
```go
// Good: Tests observable behavior
func TestEnqueueJob(t *testing.T) {
    tk := testkit.New(t, testkit.WithoutTransaction())
    tk.ResetDB()
    defer tk.ResetDB()

    svc := tk.Services.Jobs

    // Verify job executes and updates database
    executionUUID, err := svc.EnqueueJob(ctx, "test_job")
    if err != nil {
        t.Fatalf("Failed to enqueue: %v", err)
    }

    // Verify observable outcome in database
    var execution database.JobExecution
    err = tk.DB.Where("uuid = ?", executionUUID).First(&execution).Error
    // ... check status, duration, etc.
}

// Bad: Tests internal implementation
func TestInternalHandlerMap(t *testing.T) {
    svc := NewJobService(...)
    // Don't test private svc.handlers map directly
}
```

## Architecture

**Monorepo with embedded frontend**: Go backend embeds the built React frontend via `go:embed` directive in `frontend/embed.go`.

### Backend Structure
- **cmd/swopcartd/**: Application entry point, bootstraps config → database → services → server
- **internal/config/**: TOML config loading with environment variable support (`SWOPCART_DATA` for data directory)
- **internal/database/**: GORM-based PostgreSQL layer with auto-migrations; models: User, Session, Job, JobExecution, Platform, Library, Game, GameVersion
- **internal/services/**: Business logic layer with service container pattern
  - **identity/**: User identity management (no dependencies)
  - **session/**: Session and JWT token management (depends on identity)
  - **jobs/**: Background job scheduler and worker pools (depends on database)
  - **cleanup/**: Database cleanup tasks (depends on jobs)
  - **library/**: Game library management, scanning, and metadata (depends on jobs)
- **internal/www/**: Gin HTTP server and routing
- **internal/www/api/v0/**: Versioned API handlers with auth middleware
- **internal/testkit/**: Test harness with transaction-based isolation

### Frontend Structure (frontend/)
- React 19 + TypeScript + Vite + Tailwind CSS
- shadcn/ui components in `src/components/ui/`
- Pages in `src/pages/`, auth context in `src/contexts/`
- Custom hooks in `src/hooks/` for async state management

### Key Patterns
- **Config**: Loaded from `$SWOPCART_DATA/config.toml` or `./data/config.toml`; defaults in `config.DefaultConfig()`
- **Services container**: `services.Services` holds all service instances; services receive dependencies via constructor injection
- **Test harness**: `testkit.New(t)` provides isolated test environment with DB transaction rollback
- **API versioning**: Routes under `/api/v0/` with `APIHandlers.InstallRoutes()` pattern
- **JSON naming**: All JSON API responses use **camelCase** field names for consistency with JavaScript conventions (e.g., `defaultParameters`, `lastRunAt`, not `default_parameters`, `last_run_at`)
- **Logging**: Structured slog with context groups per service
- **Graceful shutdown**: Signal handling (SIGINT, SIGTERM)
- **Frontend proxy**: Dev server proxies `/api/*` to backend at :8000
- **API naming**: Frontend API functions mirror backend handler names; authorization handled server-side, not via separate "admin" functions
- **Async state management**: Custom `useAsync` and `useAsyncFn` hooks (zero dependencies) simplify Promise handling; inspired by Flutter's FutureBuilder pattern
- **Field-level errors**: `useFieldErrors` hook extracts field-specific errors from `ApiError` for form validation

### Error Handling
- **Error mapping**: Service layer errors mapped to API error codes via lookup tables (`identityErrorMap`, `sessionErrorMap` in `api/v0/error.go`)
- **Field-level errors**: Validation errors include a `key` field (e.g., `.username`, `.password`) to identify the problematic field
- **Generic error codes**: Use reusable codes like `TooShort`, `TooLong`, `InvalidChars` with field keys rather than field-specific codes
- **Security**: Session errors always return generic "Invalid token" to clients while logging actual details server-side
- **Error response format**: `{"errors": [{"error": "ErrorCode", "message": "Human message", "key": ".fieldName"}]}`

### Frontend Async Patterns
- **useAsync hook**: Auto-fetches data on mount/dependency change with automatic cleanup
  ```typescript
  const { data, loading, error } = useAsync(apiFunction, [dependencies]);
  ```
- **useAsyncFn hook**: Manual trigger for mutations/forms, returns `[state, execute, reset]` tuple
  ```typescript
  const [{ data, loading, error }, execute, reset] = useAsyncFn(mutationFunction);
  await execute(args);
  ```
- **useFieldErrors hook**: Extracts field-specific errors from `ApiError` for form validation
  ```typescript
  const fieldErrors = useFieldErrors(error);
  // Returns { username: "Too short", password: "Invalid chars" }
  ```
- **Error handling**: All components use `error.message` for display; field errors shown on specific inputs in forms
- **No caching**: Current hooks refetch on every mount; consider TanStack Query if caching/deduplication needed

### Authentication & Sessions
- **JWT structure**: Access tokens contain `sid` (session UUID), `userId` (user UUID), `username`, `admin` fields
- **Token types**: Access tokens (short-lived, ~5min) for API requests; refresh tokens (long-lived, ~90 days) for obtaining new access tokens
- **Session data**: `SessionData` struct extracted from JWT contains `SessionUUID`, `UserUUID`, `Admin` - stored in Gin context by auth middleware
- **Auth middleware**: Validates JWT, extracts session data, stores in context with key `session_data`
- **Frontend auth**: `AuthContext` provides synchronous user state; token refresh handled transparently in `fetchWithAuth`

### Background Jobs System
- **Architecture**: Cron scheduler + worker pools + queue-based task execution
- **Database models**: `Job` (metadata, schedule, config) and `JobExecution` (execution history, progress tracking)
- **Job registration**: `RegisterScheduledJob()` for cron-based jobs, `RegisterOnDemandJob()` for manual-only jobs (no schedule), `RegisterHandler()` for low-level handler registration
- **Functional options pattern**: Configure jobs with `WithPriority()`, `WithQueue()`, `WithEnabled()`, `WithDefaultParameters()`
- **Worker pools**: Per-queue pools with configurable worker count (defaults to CPU count), 100-item buffered task queue
- **Scheduler**: robfig/cron/v3 with 6-field cron expressions (second precision)
- **Progress tracking**: `ProgressReporter` interface allows real-time updates during execution
- **Parameter handling**: JSON-serialized parameters with defaults and runtime overrides
- **API endpoints**: Admin-only routes at `/api/v0/jobs/` for listing, triggering, history, and settings
- **Frontend UI**: Admin settings page (`/settings/jobs`) with job list, enable/disable toggles, manual trigger, auto-refresh
- **Graceful shutdown**: Waits for in-flight jobs with configurable timeout (default 30s)
- **Error handling**: Service errors mapped to API codes in `jobsErrorMap` (e.g., `ErrJobNotFound` → `JobNotFound`)
- **Example job**: `cleanup.soft_delete` runs daily at 3 AM to permanently delete soft-deleted records

**Registering a job:**
```go
jobSvc.RegisterScheduledJob(
    ctx,
    "example.daily_task",
    "Description of what this job does",
    "0 0 3 * * *",  // Daily at 3 AM (6-field cron: sec min hour day month weekday)
    handlerFunc,
    jobs.WithPriority(5),
    jobs.WithQueue("default"),
    jobs.WithDefaultParameters(map[string]string{"key": "value"}),
)
```

**Triggering a job manually:**
```go
executionUUID, err := jobSvc.EnqueueJob(ctx, "example.daily_task",
    jobs.WithParameters(map[string]string{"override": "value"}))
```

### Game Library Service

**Models and relationships**: `Platform` → `Library` (FK) → `Game` (FK) → `GameVersion`. Libraries have `Paths []string` (stored as JSON) for multi-directory support. Soft-delete on Library and Game; GameVersion hard-deleted on reimport.

**Scanner design** (`scan.go`): Two-pass approach — first pass counts files for progress reporting, second pass processes only top-level entries in each scan path:
- **Folder-based games**: Directory = one game. Reads `{dir}/metadata.toml` if present; auto-generates and saves it if missing.
- **Flat ROM files**: Single file = one game. Reads `{file}.toml` sidecar if present; auto-generates and saves it if missing.

**Filename metadata extraction**: Parses common ROM naming conventions from filenames/folder names:
- External IDs: `[igdb=123]`, `[vgdb=456]` → stored in `ExternalIDs` map
- Regions: `[USA]`, `[EUR]`, `[JPN]`, `.ntsc-u`, `.pal` etc.
- Tags: `(Demo)`, `(Beta)`, `(v1.0)`, `(Rev A)`

**Metadata duality**: Two representations with a conversion layer:
- **TOML on disk**: kebab-case (`release-date`, `external-ids`). Folder games use `{dir}/metadata.toml`; flat games use `{file}.toml` sidecar.
- **API JSON**: camelCase (`releaseDate`, `externalIds`). `MetadataToJSON()` / `JSONToMetadata()` convert between them.
- `GameVersion.MetadataJSON` stores the JSON string in the database for efficient API responses.

**Embedded config files pattern** (`embedded.go`): Use `//go:embed filename` to bundle default config into the binary. On service init, write to disk only if missing — allows user customization without rebuild. Example: `platforms.toml` written to `{dataDir}/platforms.toml` on first run.

**Streaming hashes**: Uses `io.MultiWriter` to compute MD5, SHA1, SHA256 in a single file pass — avoids loading ROM files into memory (critical for ARM NAS target). Hashes stored on `GameVersion`; reused on rescans if file path unchanged.

**Concurrent scan prevention**: `Library.CurrentScanJobID` (nullable UUID) set at scan start, cleared via `defer` at completion. API returns 409 if already set.

**Scan cleanup (soft-delete)**: After scanning, builds a `foundFiles` set of all discovered `FilePath` values, then soft-deletes any existing `Game` whose versions are all absent from that set.

**Library reimport**: Hard-deletes all `GameVersion` then all `Game` records for the library (bypasses soft-delete), then runs a full scan. Implemented as a separate `RegisterOnDemandJob`.

**API endpoints**:
```
GET/POST       /api/v0/libraries                              # List/create (admin)
GET/PATCH/DELETE /api/v0/libraries/{id}                      # CRUD (admin)
POST           /api/v0/libraries/{id}/scan                   # Trigger scan (admin)
POST           /api/v0/libraries/{id}/reimport               # Trigger reimport (admin)
GET            /api/v0/platforms                             # List platforms (auth)
GET            /api/v0/games                                 # Search all games (auth)
GET            /api/v0/games/by-library/{libraryId}          # Filter by library (auth)
GET            /api/v0/games/{id}                            # Game with versions (auth)
GET            /api/v0/games/{id}/versions/{vId}/download    # Stream ROM file (auth)
PATCH          /api/v0/games/{id}                            # Update metadata (admin)
```
Search endpoints support `q` (ILIKE title), `offset`, `limit` (default 50, max 100), `platformId` query params.

**Scan/reimport response**: `{"executionId": "uuid", "status": "pending"}` — client polls jobs API for progress.

**Test fixture pattern**: Scan tests copy fixture directories from `testdata/` to `t.TempDir()` for isolation, then create a `Library` record pointing to the temp path. Uses `testkit.WithoutTransaction()` + `tk.ResetDB()` because scanning involves background state. Fixture layouts:
```
testdata/flat_files/       game.zip + game.zip.toml sidecars
testdata/folder_based/     GameName/metadata.toml + rom files
testdata/mixed/            both types in one directory
```

## Tech Stack
- **Backend**: Go 1.24, Gin, GORM, PostgreSQL, robfig/cron/v3
- **Frontend**: React 19, TypeScript, Vite, Tailwind CSS, shadcn/ui, Zod
- **Testing**: Go testing with go-cmp assertions, race detector enabled
- **CI**: GitHub Actions with golangci-lint, ESLint, Prettier
- **Dev tools**: Justfile for common tasks
