package strategy

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// DBAdapter adapts sqlx.DB to our DB interface
type DBAdapter struct {
	db *sqlx.DB
}

// NewDBAdapter creates a new DB adapter
func NewDBAdapter(db *sqlx.DB) *DBAdapter {
	return &DBAdapter{db: db}
}

// BeginTx starts a new database transaction
func (a *DBAdapter) BeginTx(ctx context.Context) (Tx, error) {
	tx, err := a.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &TxAdapter{tx: tx}, nil
}

// TxAdapter adapts sqlx.Tx to our Tx interface
type TxAdapter struct {
	tx *sqlx.Tx
}

// Commit commits the transaction
func (a *TxAdapter) Commit() error {
	return a.tx.Commit()
}

// Rollback rolls back the transaction
func (a *TxAdapter) Rollback() error {
	return a.tx.Rollback()
}
