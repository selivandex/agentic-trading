package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"prometheus/internal/domain/exchange_account"
	pkgerrors "prometheus/pkg/errors"
)

// Compile-time check
var _ exchange_account.Repository = (*ExchangeAccountRepository)(nil)

// ExchangeAccountRepository implements exchange_account.Repository using sqlx
type ExchangeAccountRepository struct {
	db *sqlx.DB
}

// NewExchangeAccountRepository creates a new exchange account repository
func NewExchangeAccountRepository(db *sqlx.DB) *ExchangeAccountRepository {
	return &ExchangeAccountRepository{db: db}
}

// scanExchangeAccount scans a single exchange account from database row
func scanExchangeAccount(row interface {
	Scan(dest ...interface{}) error
}) (*exchange_account.ExchangeAccount, error) {
	acc := &exchange_account.ExchangeAccount{}

	err := row.Scan(
		&acc.ID, &acc.UserID, &acc.Exchange, &acc.Label,
		&acc.APIKeyEncrypted, &acc.SecretEncrypted,
		&acc.Passphrase, &acc.IsTestnet, pq.Array(&acc.Permissions),
		&acc.IsActive, &acc.LastSyncAt, &acc.CreatedAt, &acc.UpdatedAt,
		&acc.ListenKeyEncrypted, &acc.ListenKeyExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	return acc, nil
}

// Create inserts a new exchange account
// Note: API keys should already be encrypted before calling this
func (r *ExchangeAccountRepository) Create(ctx context.Context, account *exchange_account.ExchangeAccount) error {
	query := `
		INSERT INTO exchange_accounts (
			id, user_id, exchange, label, api_key_encrypted, secret_encrypted,
			passphrase, is_testnet, permissions, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)`

	_, err := r.db.ExecContext(ctx, query,
		account.ID, account.UserID, account.Exchange, account.Label,
		account.APIKeyEncrypted, account.SecretEncrypted,
		account.Passphrase, account.IsTestnet, pq.Array(account.Permissions),
		account.IsActive, account.CreatedAt, account.UpdatedAt,
	)

	return err
}

// GetByID retrieves an exchange account by ID
func (r *ExchangeAccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*exchange_account.ExchangeAccount, error) {
	query := `
		SELECT 
			id, user_id, exchange, label,
			api_key_encrypted, secret_encrypted,
			passphrase, is_testnet, permissions,
			is_active, last_sync_at, created_at, updated_at,
			listen_key_encrypted, listen_key_expires_at
		FROM exchange_accounts 
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)
	account, err := scanExchangeAccount(row)
	if err == sql.ErrNoRows {
		return nil, pkgerrors.Wrap(err, "exchange account not found")
	}
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to get exchange account")
	}

	return account, nil
}

// GetByUser retrieves all exchange accounts for a user
func (r *ExchangeAccountRepository) GetByUser(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error) {
	query := `
		SELECT 
			id, user_id, exchange, label,
			api_key_encrypted, secret_encrypted,
			passphrase, is_testnet, permissions,
			is_active, last_sync_at, created_at, updated_at,
			listen_key_encrypted, listen_key_expires_at
		FROM exchange_accounts 
		WHERE user_id = $1 
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to query exchange accounts")
	}
	defer rows.Close()

	var accounts []*exchange_account.ExchangeAccount
	for rows.Next() {
		acc, err := scanExchangeAccount(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan exchange account")
		}
		accounts = append(accounts, acc)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate exchange accounts")
	}

	return accounts, nil
}

// GetActiveByUser retrieves active exchange accounts for a user
func (r *ExchangeAccountRepository) GetActiveByUser(ctx context.Context, userID uuid.UUID) ([]*exchange_account.ExchangeAccount, error) {
	query := `
		SELECT 
			id, user_id, exchange, label,
			api_key_encrypted, secret_encrypted,
			passphrase, is_testnet, permissions,
			is_active, last_sync_at, created_at, updated_at,
			listen_key_encrypted, listen_key_expires_at
		FROM exchange_accounts 
		WHERE user_id = $1 AND is_active = true 
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to query active exchange accounts")
	}
	defer rows.Close()

	var accounts []*exchange_account.ExchangeAccount
	for rows.Next() {
		acc, err := scanExchangeAccount(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan exchange account")
		}
		accounts = append(accounts, acc)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate exchange accounts")
	}

	return accounts, nil
}

// GetAllActive retrieves all active exchange accounts across all users
// Used by UserDataManager for hot reload and reconciliation
func (r *ExchangeAccountRepository) GetAllActive(ctx context.Context) ([]*exchange_account.ExchangeAccount, error) {
	query := `
		SELECT 
			id, user_id, exchange, label,
			api_key_encrypted, secret_encrypted,
			passphrase, is_testnet, permissions,
			is_active, last_sync_at, created_at, updated_at,
			listen_key_encrypted, listen_key_expires_at
		FROM exchange_accounts 
		WHERE is_active = true 
		ORDER BY user_id, created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to query all active exchange accounts")
	}
	defer rows.Close()

	var accounts []*exchange_account.ExchangeAccount
	for rows.Next() {
		acc, err := scanExchangeAccount(rows)
		if err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan exchange account")
		}
		accounts = append(accounts, acc)
	}

	if err := rows.Err(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to iterate all active exchange accounts")
	}

	return accounts, nil
}

// Update updates an exchange account
func (r *ExchangeAccountRepository) Update(ctx context.Context, account *exchange_account.ExchangeAccount) error {
	query := `
		UPDATE exchange_accounts SET
			label = $2,
			permissions = $3,
			is_active = $4,
			listen_key_encrypted = $5,
			listen_key_expires_at = $6,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		account.ID, account.Label, pq.Array(account.Permissions), account.IsActive,
		account.ListenKeyEncrypted, account.ListenKeyExpiresAt,
	)

	return err
}

// UpdateLastSync updates the last sync timestamp
func (r *ExchangeAccountRepository) UpdateLastSync(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE exchange_accounts SET last_sync_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Delete deletes an exchange account
func (r *ExchangeAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM exchange_accounts WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
