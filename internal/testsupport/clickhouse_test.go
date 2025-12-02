package testsupport

import (
	"context"
	"testing"
)

func TestClickHouseCleanupDropsTable(t *testing.T) {
	configs := LoadDatabaseConfigsFromEnv(t)

	helper := NewClickHouseTestHelper(t, configs.ClickHouse)
	table := helper.CreateTempTable(t, "id UInt64, value String")

	if err := helper.Client().Exec(context.Background(), "INSERT INTO "+table+" (id, value) VALUES (1, 'abc')"); err != nil {
		t.Fatalf("failed to insert row: %v", err)
	}

	var counts []uint64
	if err := helper.Client().Query(context.Background(), &counts, "SELECT count() FROM "+table); err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if len(counts) == 0 {
		t.Fatal("no count returned")
	}

	count := counts[0]

	if count != 1 {
		t.Fatalf("unexpected row count: %d", count)
	}

	if err := helper.CleanupTable(context.Background(), table); err != nil {
		t.Fatalf("failed to cleanup table: %v", err)
	}

	var exists int
	if err := helper.Client().Query(context.Background(), &exists, "EXISTS TABLE "+table); err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}

	if exists != 0 {
		t.Fatalf("expected table to be dropped, exists=%d", exists)
	}
}
