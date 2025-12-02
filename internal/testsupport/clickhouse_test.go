package testsupport

import (
	"context"
	"testing"
)

func TestClickHouseCleanupDropsTable(t *testing.T) {
	helper := NewClickHouseTestHelper(t, GetConfig().ClickHouse)
	table := helper.CreateTempTable(t, "id UInt64, value String")

	if err := helper.Client().Exec(context.Background(), "INSERT INTO "+table+" (id, value) VALUES (1, 'abc')"); err != nil {
		t.Fatalf("failed to insert row: %v", err)
	}

	// Query count using row scan
	var count uint64
	row := helper.Client().Conn().QueryRow(context.Background(), "SELECT count() FROM "+table)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if count != 1 {
		t.Fatalf("unexpected row count: %d", count)
	}

	if err := helper.CleanupTable(context.Background(), table); err != nil {
		t.Fatalf("failed to cleanup table: %v", err)
	}

	// Check if table exists
	var exists uint8
	row = helper.Client().Conn().QueryRow(context.Background(), "EXISTS TABLE "+table)
	if err := row.Scan(&exists); err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}

	if exists != 0 {
		t.Fatalf("expected table to be dropped, exists=%d", exists)
	}
}
