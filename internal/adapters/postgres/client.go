package postgres

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver

	"prometheus/internal/adapters/config"
	"prometheus/pkg/errors"
)

// Client wraps sqlx.DB for PostgreSQL operations
type Client struct {
	db *sqlx.DB
}

// NewClient creates a new PostgreSQL client with connection pooling
func NewClient(cfg config.PostgresConfig) (*Client, error) {
	db, err := sqlx.Connect("postgres", cfg.DSN())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to postgres")
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns / 2)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	// Verify connection
	if err := db.PingContext(context.Background()); err != nil {
		return nil, errors.Wrap(err, "failed to ping postgres")
	}

	return &Client{db: db}, nil
}

// DB returns the underlying sqlx.DB instance
func (c *Client) DB() *sqlx.DB {
	return c.db
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.db.Close()
}

// Health checks database connectivity
func (c *Client) Health(ctx context.Context) error {
	return c.db.PingContext(ctx)
}
