# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
```

### Testing
Tests require PostgreSQL. Configure via `SWOPCART_TEST_DB` env var or use default:
```bash
# Default: postgres://swopcart:swopcart@localhost:5432/swopcart_test
SWOPCART_TEST_DB="postgres://user:pass@host:5432/dbname" go test -v -race ./...
```

## Architecture

**Monorepo with embedded frontend**: Go backend embeds the built React frontend via `go:embed` directive in `frontend/embed.go`.

### Backend Structure
- **cmd/swopcartd/**: Application entry point, bootstraps config → database → services → server
- **internal/config/**: TOML config loading with environment variable support (`SWOPCART_DATA` for data directory)
- **internal/database/**: GORM-based PostgreSQL layer with auto-migrations; models: User, Session
- **internal/services/**: Business logic layer with service container pattern
  - **identity/**: User identity management (no dependencies)
  - **session/**: Session and JWT token management (depends on identity)
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

## Tech Stack
- **Backend**: Go 1.24, Gin, GORM, PostgreSQL
- **Frontend**: React 19, TypeScript, Vite, Tailwind CSS, shadcn/ui
- **Testing**: Go testing with go-cmp assertions, race detector enabled
- **CI**: GitHub Actions with golangci-lint, ESLint, Prettier
