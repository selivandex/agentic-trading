package testsupport

import (
	"context"
	"database/sql"
	"testing"
)

func TestPostgresTransactionIsRolledBack(t *testing.T) {
	configs := LoadDatabaseConfigsFromEnv(t)

	helper := NewPostgresTestHelper(t, configs.Postgres)
	tx := helper.Tx()

	// Create table and insert inside the transaction
	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS integration_tx_check(id SERIAL PRIMARY KEY, value TEXT)"); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	if _, err := tx.Exec("INSERT INTO integration_tx_check(value) VALUES('hello world')"); err != nil {
		t.Fatalf("failed to insert row: %v", err)
	}

	var count int
	if err := tx.QueryRow("SELECT COUNT(*) FROM integration_tx_check").Scan(&count); err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if count != 1 {
		t.Fatalf("unexpected row count inside transaction: %d", count)
	}

	helper.Rollback()

	// Verify the table does not exist after rollback
	var exists sql.NullString
	err := helper.DB().QueryRowContext(context.Background(), "SELECT to_regclass('public.integration_tx_check')").Scan(&exists)
	if err != nil {
		t.Fatalf("failed to query table existence: %v", err)
	}

	if exists.Valid {
		t.Fatalf("expected table to be rolled back, found: %s", exists.String)
	}
}
