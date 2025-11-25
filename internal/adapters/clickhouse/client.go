package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"prometheus/internal/adapters/config"
)

// Client wraps ClickHouse connection
type Client struct {
	conn driver.Conn
}

// NewClient creates a new ClickHouse client
func NewClient(cfg config.ClickHouseConfig) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	// Verify connection
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return &Client{conn: conn}, nil
}

// Conn returns the underlying ClickHouse connection
func (c *Client) Conn() driver.Conn {
	return c.conn
}

// Close closes the ClickHouse connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// Health checks ClickHouse connectivity
func (c *Client) Health(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Exec executes a query without returning rows
func (c *Client) Exec(ctx context.Context, query string, args ...interface{}) error {
	return c.conn.Exec(ctx, query, args...)
}

// Query executes a query and returns rows
func (c *Client) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return c.conn.Select(ctx, dest, query, args...)
}

// AsyncInsert performs async insert (faster for batch inserts)
func (c *Client) AsyncInsert(ctx context.Context, query string, data interface{}) error {
	batch, err := c.conn.PrepareBatch(ctx, query)
	if err != nil {
		return err
	}

	if err := batch.AppendStruct(data); err != nil {
		return err
	}

	return batch.Send()
}
