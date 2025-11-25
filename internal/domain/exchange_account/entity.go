package exchange_account

import (
	"time"

	"github.com/google/uuid"
)

// ExchangeAccount represents a user's exchange connection
type ExchangeAccount struct {
	ID              uuid.UUID    `db:"id"`
	UserID          uuid.UUID    `db:"user_id"`
	Exchange        ExchangeType `db:"exchange"` // binance, bybit, okx
	Label           string       `db:"label"`    // "Main Binance", etc.
	APIKeyEncrypted []byte       `db:"api_key_encrypted"`
	SecretEncrypted []byte       `db:"secret_encrypted"`
	Passphrase      []byte       `db:"passphrase"` // For OKX
	IsTestnet       bool         `db:"is_testnet"`
	Permissions     []string     `db:"permissions"` // spot, futures, read, trade
	IsActive        bool         `db:"is_active"`
	LastSyncAt      *time.Time   `db:"last_sync_at"`
	CreatedAt       time.Time    `db:"created_at"`
	UpdatedAt       time.Time    `db:"updated_at"`
}

// ExchangeType defines supported exchanges
type ExchangeType string

const (
	ExchangeBinance ExchangeType = "binance"
	ExchangeBybit   ExchangeType = "bybit"
	ExchangeOKX     ExchangeType = "okx"
	ExchangeKucoin  ExchangeType = "kucoin"
	ExchangeGate    ExchangeType = "gate"
)

// Valid checks if exchange type is valid
func (e ExchangeType) Valid() bool {
	switch e {
	case ExchangeBinance, ExchangeBybit, ExchangeOKX, ExchangeKucoin, ExchangeGate:
		return true
	}
	return false
}

// String returns string representation
func (e ExchangeType) String() string {
	return string(e)
}

import (
	"time"

	"github.com/google/uuid"
)

// ExchangeAccount represents a user's exchange connection
type ExchangeAccount struct {
	ID              uuid.UUID    `db:"id"`
	UserID          uuid.UUID    `db:"user_id"`
	Exchange        ExchangeType `db:"exchange"` // binance, bybit, okx
	Label           string       `db:"label"`    // "Main Binance", etc.
	APIKeyEncrypted []byte       `db:"api_key_encrypted"`
	SecretEncrypted []byte       `db:"secret_encrypted"`
	Passphrase      []byte       `db:"passphrase"` // For OKX
	IsTestnet       bool         `db:"is_testnet"`
	Permissions     []string     `db:"permissions"` // spot, futures, read, trade
	IsActive        bool         `db:"is_active"`
	LastSyncAt      *time.Time   `db:"last_sync_at"`
	CreatedAt       time.Time    `db:"created_at"`
	UpdatedAt       time.Time    `db:"updated_at"`
}

// ExchangeType defines supported exchanges
type ExchangeType string

const (
	ExchangeBinance ExchangeType = "binance"
	ExchangeBybit   ExchangeType = "bybit"
	ExchangeOKX     ExchangeType = "okx"
	ExchangeKucoin  ExchangeType = "kucoin"
	ExchangeGate    ExchangeType = "gate"
)

// Valid checks if exchange type is valid
func (e ExchangeType) Valid() bool {
	switch e {
	case ExchangeBinance, ExchangeBybit, ExchangeOKX, ExchangeKucoin, ExchangeGate:
		return true
	}
	return false
}

// String returns string representation
func (e ExchangeType) String() string {
	return string(e)
}
