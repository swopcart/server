package testkit

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services"
	"gorm.io/gorm"
)

// Harness provides a test environment with database, services, and HTTP server.
// The database runs within a transaction that is rolled back when the test completes,
// ensuring test isolation.
type Harness struct {
	T        *testing.T
	Config   *config.Config
	Logger   *slog.Logger
	DB       *gorm.DB
	Services *services.Services
	Router   *gin.Engine
}

// HarnessOption configures the test harness.
type HarnessOption func(*harnessConfig)

type harnessConfig struct {
	useTransaction bool
}

// WithoutTransaction disables transaction-based isolation.
// Use this for tests that need committed data (e.g., testing async workers).
// When using this option, tests should call ResetDB() before and after to ensure isolation:
//
//	tk := testkit.New(t, testkit.WithoutTransaction())
//	tk.ResetDB()         // Clean to known state
//	defer tk.ResetDB()   // Clean up after test
func WithoutTransaction() HarnessOption {
	return func(cfg *harnessConfig) {
		cfg.useTransaction = false
	}
}

// New creates a new test harness. It reads the database connection string from
// the SWOPCART_TEST_DB environment variable.
//
// By default, tests run within a transaction that is rolled back on completion.
// Use WithoutTransaction() for tests that need committed data.
func New(t *testing.T, opts ...HarnessOption) *Harness {
	t.Helper()

	// Parse options
	hCfg := &harnessConfig{
		useTransaction: true, // Default to transaction-based isolation
	}
	for _, opt := range opts {
		opt(hCfg)
	}

	connStr := os.Getenv("SWOPCART_TEST_DB")
	if connStr == "" {
		connStr = "postgres://swopcart:swopcart@localhost:5432/swopcart_test"
	}

	// Create a data directory in the temp dir for test files
	dataDir := t.TempDir() + "/data"
	if err := os.Mkdir(dataDir, 0755); err != nil {
		t.Fatalf("failed to create test data directory: %v", err)
	}
	t.Setenv("SWOPCART_DATA", dataDir)

	cfg := &config.Config{
		Server: config.Server{
			Network: "tcp",
			Address: "127.0.0.1:0",
		},
		Database: config.Database{
			Conn: connStr,
		},
		Auth: config.Auth{
			KeyPath:         "auth.key",
			AccessTokenTTL:  300, // 5 minutes
			RefreshTokenTTL: 90,  // 90 days
			JWTIssuer:       "swopcart-test",
		},
		Jobs: config.Jobs{
			DefaultWorkers:  2, // Minimal workers for tests
			ShutdownTimeout: 5 * time.Second,
			Queues:          make(map[string]config.QueueConfig),
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	db, err := database.NewDatabase(cfg, logger)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	var dbConn *gorm.DB
	if hCfg.useTransaction {
		// Start a transaction that will be rolled back at the end of the test.
		// This ensures each test runs in isolation with a clean database state.
		tx := db.Begin()
		if tx.Error != nil {
			t.Fatalf("failed to begin transaction: %v", tx.Error)
		}

		t.Cleanup(func() {
			tx.Rollback()
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				_ = sqlDB.Close()
			}
		})

		dbConn = tx
	} else {
		// No transaction - tests must manually clean up data.
		// Acquire a session advisory lock so that only one WithoutTransaction test
		// runs at a time across all packages. This prevents ResetDB (TRUNCATE) calls
		// in one package from deleting committed data that another package's concurrent
		// test depends on.
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("failed to get sql.DB for advisory lock: %v", err)
		}
		lockConn, err := sqlDB.Conn(context.Background())
		if err != nil {
			t.Fatalf("failed to open advisory lock connection: %v", err)
		}
		if _, err := lockConn.ExecContext(context.Background(), "SELECT pg_advisory_lock(1234567890)"); err != nil {
			_ = lockConn.Close()
			t.Fatalf("failed to acquire test exclusion lock: %v", err)
		}

		t.Cleanup(func() {
			_, _ = lockConn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(1234567890)")
			_ = lockConn.Close()
			sqlDB2, _ := db.DB()
			if sqlDB2 != nil {
				_ = sqlDB2.Close()
			}
		})

		dbConn = db
	}

	ctx := context.Background()

	// Services use the transaction (or raw db if no transaction)
	svcs, err := services.NewServices(ctx, cfg, logger, dbConn)
	if err != nil {
		t.Fatalf("failed to create services: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	return &Harness{
		T:        t,
		Config:   cfg,
		Logger:   logger,
		DB:       dbConn,
		Services: svcs,
		Router:   router,
	}
}

// Request performs an HTTP request against the test router and returns the response.
func (h *Harness) Request(method, path string, body io.Reader) *httptest.ResponseRecorder {
	h.T.Helper()

	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.Router.ServeHTTP(rec, req)

	return rec
}

// GET performs a GET request against the test router.
func (h *Harness) GET(path string) *httptest.ResponseRecorder {
	return h.Request(http.MethodGet, path, nil)
}

// POST performs a POST request against the test router.
func (h *Harness) POST(path string, body io.Reader) *httptest.ResponseRecorder {
	return h.Request(http.MethodPost, path, body)
}

// PUT performs a PUT request against the test router.
func (h *Harness) PUT(path string, body io.Reader) *httptest.ResponseRecorder {
	return h.Request(http.MethodPut, path, body)
}

// DELETE performs a DELETE request against the test router.
func (h *Harness) DELETE(path string) *httptest.ResponseRecorder {
	return h.Request(http.MethodDelete, path, nil)
}

// ResetDB removes all data from all tables, resetting the database to an empty state.
// This is useful for tests that use WithoutTransaction() and need to clean up between tests.
// Tables are truncated in an order that respects foreign key constraints.
func (h *Harness) ResetDB() {
	h.T.Helper()

	models := func() (m []any) {
		m = slices.Clone(database.Models)
		slices.Reverse(m)
		return
	}()

	for _, model := range models {
		table := func() string {
			stmt := &gorm.Statement{DB: h.DB}
			if err := stmt.Parse(model); err != nil {
				h.T.Logf("can't get table name: %v", err)
				h.T.FailNow()
			}

			return stmt.Schema.Table
		}()

		if err := h.DB.Exec("TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE").Error; err != nil {
			h.T.Fatalf("failed to truncate table %s: %v", table, err)
		}
	}
}
