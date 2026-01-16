# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

### Backend (Go)
```bash
go build -v ./...                                    # Build
go test -v -race ./...                              # Run tests
go test -v ./config                                 # Test specific package
golangci-lint run --timeout=5m                      # Lint
go generate -v ./...                                # Rebuild embedded frontend assets
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

## Architecture

**Monorepo with embedded frontend**: Go backend embeds the built React frontend via `go:embed` directive in `frontend/embed.go`.

### Backend Structure
- **cmd/swopcartd/**: Application entry point, bootstraps config, database, and server
- **config/**: TOML config loading with environment variable support (`SWOPCART_DATA` for data directory)
- **database/**: GORM-based PostgreSQL layer with auto-migrations; models: User, Session
- **www/**: Gin HTTP server and routing
- **www/api/v0/**: Versioned API handlers (v0 namespace)
- **internal/testkit/**: Test utilities

### Frontend Structure (frontend/)
- React 19 + TypeScript + Vite + Tailwind CSS
- shadcn/ui components in `src/components/ui/`
- Pages in `src/pages/`

### Key Patterns
- Config loaded from `$SWOPCART_DATA/config.toml` or `./data/config.toml`
- Structured logging via slog with context groups
- Graceful shutdown with signal handling (SIGINT, SIGTERM)
- API routes under `/api/v0/` with versioning support
- Frontend dev server proxies `/api/*` to backend at :8000

## Tech Stack
- **Backend**: Go 1.24, Gin, GORM, PostgreSQL
- **Frontend**: React 19, TypeScript, Vite, Tailwind CSS, shadcn/ui
- **Testing**: Go testing with go-cmp assertions, race detector enabled
- **CI**: GitHub Actions with golangci-lint, ESLint, Prettier
