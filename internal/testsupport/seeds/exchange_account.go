package seeds

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"prometheus/internal/domain/exchange_account"
)

// ExchangeAccountBuilder provides a fluent API for creating ExchangeAccount entities
type ExchangeAccountBuilder struct {
	db     DBTX
	ctx    context.Context
	entity *exchange_account.ExchangeAccount
}

// NewExchangeAccountBuilder creates a new ExchangeAccountBuilder with sensible defaults
func NewExchangeAccountBuilder(db DBTX, ctx context.Context) *ExchangeAccountBuilder {
	now := time.Now()
	return &ExchangeAccountBuilder{
		db:  db,
		ctx: ctx,
		entity: &exchange_account.ExchangeAccount{
			ID:              uuid.New(),
			UserID:          uuid.Nil, // Must be set
			Exchange:        exchange_account.ExchangeBinance,
			Label:           "Test Account",
			APIKeyEncrypted: []byte("encrypted_key"),
			SecretEncrypted: []byte("encrypted_secret"),
			Passphrase:      nil,
			IsTestnet:       true,
			Permissions:     []string{"spot", "read", "trade"},
			IsActive:        true,
			LastSyncAt:      nil,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}
}

// WithID sets a specific ID
func (b *ExchangeAccountBuilder) WithID(id uuid.UUID) *ExchangeAccountBuilder {
	b.entity.ID = id
	return b
}

// WithUserID sets the user ID (required)
func (b *ExchangeAccountBuilder) WithUserID(userID uuid.UUID) *ExchangeAccountBuilder {
	b.entity.UserID = userID
	return b
}

// WithExchange sets the exchange type
func (b *ExchangeAccountBuilder) WithExchange(exchange exchange_account.ExchangeType) *ExchangeAccountBuilder {
	b.entity.Exchange = exchange
	return b
}

// WithBinance is a convenience method to set exchange to Binance
func (b *ExchangeAccountBuilder) WithBinance() *ExchangeAccountBuilder {
	b.entity.Exchange = exchange_account.ExchangeBinance
	return b
}

// WithBybit is a convenience method to set exchange to Bybit
func (b *ExchangeAccountBuilder) WithBybit() *ExchangeAccountBuilder {
	b.entity.Exchange = exchange_account.ExchangeBybit
	return b
}

// WithOKX is a convenience method to set exchange to OKX
func (b *ExchangeAccountBuilder) WithOKX() *ExchangeAccountBuilder {
	b.entity.Exchange = exchange_account.ExchangeOKX
	return b
}

// WithLabel sets the account label
func (b *ExchangeAccountBuilder) WithLabel(label string) *ExchangeAccountBuilder {
	b.entity.Label = label
	return b
}

// WithTestnet sets whether this is a testnet account
func (b *ExchangeAccountBuilder) WithTestnet(isTestnet bool) *ExchangeAccountBuilder {
	b.entity.IsTestnet = isTestnet
	return b
}

// WithActive sets the active status
func (b *ExchangeAccountBuilder) WithActive(active bool) *ExchangeAccountBuilder {
	b.entity.IsActive = active
	return b
}

// WithPermissions sets the permissions
func (b *ExchangeAccountBuilder) WithPermissions(permissions []string) *ExchangeAccountBuilder {
	b.entity.Permissions = permissions
	return b
}

// WithAPIKey sets encrypted API key (for testing, use simple value)
func (b *ExchangeAccountBuilder) WithAPIKey(key []byte) *ExchangeAccountBuilder {
	b.entity.APIKeyEncrypted = key
	return b
}

// WithSecret sets encrypted secret (for testing, use simple value)
func (b *ExchangeAccountBuilder) WithSecret(secret []byte) *ExchangeAccountBuilder {
	b.entity.SecretEncrypted = secret
	return b
}

// WithPassphrase sets encrypted passphrase (for OKX)
func (b *ExchangeAccountBuilder) WithPassphrase(passphrase []byte) *ExchangeAccountBuilder {
	b.entity.Passphrase = passphrase
	return b
}

// WithLastSyncAt sets the last sync timestamp
func (b *ExchangeAccountBuilder) WithLastSyncAt(t time.Time) *ExchangeAccountBuilder {
	b.entity.LastSyncAt = &t
	return b
}

// Build returns the built entity without inserting to DB
func (b *ExchangeAccountBuilder) Build() *exchange_account.ExchangeAccount {
	return b.entity
}

// Insert inserts the exchange account into the database and returns the entity
func (b *ExchangeAccountBuilder) Insert() (*exchange_account.ExchangeAccount, error) {
	if b.entity.UserID == uuid.Nil {
		return nil, fmt.Errorf("user_id is required")
	}

	query := `
		INSERT INTO exchange_accounts (
			id, user_id, exchange, label, api_key_encrypted, secret_encrypted,
			passphrase, is_testnet, permissions, is_active, last_sync_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := b.db.ExecContext(
		b.ctx,
		query,
		b.entity.ID,
		b.entity.UserID,
		b.entity.Exchange,
		b.entity.Label,
		b.entity.APIKeyEncrypted,
		b.entity.SecretEncrypted,
		b.entity.Passphrase,
		b.entity.IsTestnet,
		b.entity.Permissions,
		b.entity.IsActive,
		b.entity.LastSyncAt,
		b.entity.CreatedAt,
		b.entity.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert exchange account: %w", err)
	}

	return b.entity, nil
}

// MustInsert inserts the exchange account and panics on error (useful for tests)
func (b *ExchangeAccountBuilder) MustInsert() *exchange_account.ExchangeAccount {
	entity, err := b.Insert()
	if err != nil {
		panic(err)
	}
	return entity
}


