package testsupport

import (
	"context"
	"fmt"
	"testing"
	"time"

	"prometheus/internal/adapters/clickhouse"
	"prometheus/internal/adapters/config"
)

// ClickHouseTestHelper manages cleanup for ClickHouse integration tests.
type ClickHouseTestHelper struct {
	client *clickhouse.Client
}

// NewClickHouseTestHelper creates a ClickHouse client for tests.
func NewClickHouseTestHelper(t *testing.T, cfg config.ClickHouseConfig) *ClickHouseTestHelper {
	t.Helper()

	client, err := clickhouse.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect to clickhouse: %v", err)
	}

	helper := &ClickHouseTestHelper{client: client}
	t.Cleanup(func() { _ = client.Close() })
	return helper
}

// CreateTempTable creates a temporary table and registers cleanup.
func (h *ClickHouseTestHelper) CreateTempTable(t *testing.T, schema string) string {
	t.Helper()

	table := fmt.Sprintf("tmp_test_%d", time.Now().UnixNano())
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s) ENGINE = MergeTree() ORDER BY tuple()", table, schema)

	if err := h.client.Exec(context.Background(), query); err != nil {
		t.Fatalf("failed to create clickhouse table: %v", err)
	}

	t.Cleanup(func() {
		_ = h.client.Exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	})

	return table
}

// CleanupTable drops the provided table immediately.
func (h *ClickHouseTestHelper) CleanupTable(ctx context.Context, table string) error {
	return h.client.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
}

// Client exposes the raw ClickHouse client for queries.
func (h *ClickHouseTestHelper) Client() *clickhouse.Client {
	return h.client
}
