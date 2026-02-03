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
Foundation complete: authentication, user management, session handling, background jobs system. Ready to build core game library features.

## Active Development

### Recently Completed (2026-02-03)
- **Background job service**: ✓ Complete
  - Cron-based scheduler with worker pools and queue management
  - Admin UI for job management, triggering, and monitoring
  - Progress tracking for long-running tasks
  - Database cleanup maintenance job implemented as first example

### Next Steps
- Game library domain model (Platforms, Games, ROM files, User libraries)
- ROM file scanning and identification (hash-based lookups)
- Metadata scraping/integration (IGDB or similar)
- Save file sync system with conflict resolution
- Client download/streaming endpoints

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
- **internal/database/**: GORM-based PostgreSQL layer with auto-migrations; models: User, Session, Job, JobExecution
- **internal/services/**: Business logic layer with service container pattern
  - **identity/**: User identity management (no dependencies)
  - **session/**: Session and JWT token management (depends on identity)
  - **jobs/**: Background job scheduler and worker pools (depends on database)
  - **cleanup/**: Database cleanup tasks (depends on jobs)
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
- **Job registration**: `RegisterScheduledJob()` for cron-based jobs, `RegisterHandler()` for on-demand jobs
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

## Tech Stack
- **Backend**: Go 1.24, Gin, GORM, PostgreSQL, robfig/cron/v3
- **Frontend**: React 19, TypeScript, Vite, Tailwind CSS, shadcn/ui, Zod
- **Testing**: Go testing with go-cmp assertions, race detector enabled
- **CI**: GitHub Actions with golangci-lint, ESLint, Prettier
- **Dev tools**: Justfile for common tasks
