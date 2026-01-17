package testkit

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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

// New creates a new test harness. It reads the database connection string from
// the SWOPCART_TEST_DB environment variable.
func New(t *testing.T) *Harness {
	t.Helper()

	connStr := os.Getenv("SWOPCART_TEST_DB")
	if connStr == "" {
		connStr = "postgres://swopcart:swopcart@localhost:5432/swopcart_test"
	}

	cfg := &config.Config{
		Server: config.Server{
			Network: "tcp",
			Address: "127.0.0.1:0",
		},
		Database: config.Database{
			Conn: connStr,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	db, err := database.NewDatabase(cfg, logger)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

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

	ctx := context.Background()

	// Services use the transaction, not the raw db connection
	svcs, err := services.NewServices(ctx, cfg, logger, tx)
	if err != nil {
		t.Fatalf("failed to create services: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	return &Harness{
		T:        t,
		Config:   cfg,
		Logger:   logger,
		DB:       tx, // expose the transaction as DB so all operations use it
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
