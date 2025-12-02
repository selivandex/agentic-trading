package postgres

import (
	"context"
	"database/sql"
)

// DBTX is a common interface for *sqlx.DB and *sqlx.Tx
// This allows repositories to work with both regular connections and transactions
// enabling full transactional isolation in tests
type DBTX interface {
	// Core query methods
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row

	// sqlx extended methods
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error

	// Named query support
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
}
