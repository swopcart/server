package jobs

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/database"
	"gorm.io/gorm"
)

// testHarness provides a minimal test environment for jobs tests
// without creating a circular dependency with testkit/services
type testHarness struct {
	t      *testing.T
	config *config.Config
	logger *slog.Logger
	db     *gorm.DB
	ctx    context.Context
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	connStr := os.Getenv("SWOPCART_TEST_DB")
	if connStr == "" {
		connStr = "postgres://swopcart:swopcart@localhost:5432/swopcart_test"
	}

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
			AccessTokenTTL:  300,
			RefreshTokenTTL: 90,
			JWTIssuer:       "swopcart-test",
		},
		Jobs: config.Jobs{
			DefaultWorkers:  2,
			ShutdownTimeout: 5000000000, // 5 seconds in nanoseconds
			Queues:          make(map[string]config.QueueConfig),
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	db, err := database.NewDatabase(cfg, logger)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Clean up any committed data from previous test runs
	// This ensures tests don't see leftover data when using transactions
	tables := []string{"job_executions", "jobs", "sessions", "users"}
	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			// Ignore errors for tables that don't exist
			t.Logf("warning: failed to clean table %s: %v", table, err)
		}
	}

	// Start a transaction for isolation
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

	return &testHarness{
		t:      t,
		config: cfg,
		logger: logger,
		db:     tx,
		ctx:    context.Background(),
	}
}

func (h *testHarness) Context() context.Context {
	return h.ctx
}

func (h *testHarness) Config() *config.Config {
	return h.config
}

func (h *testHarness) Logger() *slog.Logger {
	return h.logger
}

func (h *testHarness) DB() *gorm.DB {
	return h.db
}

func (h *testHarness) UUID() uuid.UUID {
	return uuid.New()
}
