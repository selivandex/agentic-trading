package testsupport

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"

	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/postgres"
)

// PostgresTestHelper manages a transactional connection for integration tests.
type PostgresTestHelper struct {
	client     *postgres.Client
	tx         *sqlx.Tx
	rolledBack bool
}

// NewPostgresTestHelper opens a connection and begins a transaction that is always rolled back.
func NewPostgresTestHelper(t *testing.T, cfg config.PostgresConfig) *PostgresTestHelper {
	t.Helper()

	client, err := postgres.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create postgres client: %v", err)
	}

	tx, err := client.DB().BeginTxx(context.Background(), nil)
	if err != nil {
		_ = client.Close()
		t.Fatalf("failed to start transaction: %v", err)
	}

	helper := &PostgresTestHelper{client: client, tx: tx}
	t.Cleanup(helper.Rollback)
	t.Cleanup(func() {
		_ = client.Close()
	})

	return helper
}

// Tx returns the active transaction for the test.
func (h *PostgresTestHelper) Tx() *sqlx.Tx {
	return h.tx
}

// DB returns the underlying database handle.
func (h *PostgresTestHelper) DB() *sqlx.DB {
	return h.client.DB()
}

// Rollback rolls back the transaction once.
func (h *PostgresTestHelper) Rollback() {
	if h.rolledBack {
		return
	}
	_ = h.tx.Rollback()
	h.rolledBack = true
}

// Close is an alias for Rollback for backward compatibility
func (h *PostgresTestHelper) Close() {
	h.Rollback()
}

// NewTestPostgres creates a test postgres helper with config loaded from .env.test
// This is a convenience wrapper for NewPostgresTestHelper with default test config
func NewTestPostgres(t *testing.T) *PostgresTestHelper {
	t.Helper()

	// Load database configs from .env.test (uses godotenv internally)
	dbConfigs := LoadDatabaseConfigsFromEnv(t)

	return NewPostgresTestHelper(t, dbConfigs.Postgres)
}
