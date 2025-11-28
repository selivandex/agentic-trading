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

// Encryptor interface for encryption/decryption operations
// This allows domain to be independent from crypto implementation
type Encryptor interface {
	Encrypt(plaintext string) ([]byte, error)
	Decrypt(ciphertext []byte) (string, error)
}

// SetAPIKey encrypts and sets the API key
func (ea *ExchangeAccount) SetAPIKey(plaintext string, encryptor Encryptor) error {
	if encryptor == nil {
		return nil // Skip encryption if no encryptor provided
	}

	encrypted, err := encryptor.Encrypt(plaintext)
	if err != nil {
		return err
	}

	ea.APIKeyEncrypted = encrypted
	return nil
}

// GetAPIKey decrypts and returns the API key
func (ea *ExchangeAccount) GetAPIKey(encryptor Encryptor) (string, error) {
	if encryptor == nil || len(ea.APIKeyEncrypted) == 0 {
		return "", nil
	}

	plaintext, err := encryptor.Decrypt(ea.APIKeyEncrypted)
	if err != nil {
		return "", err
	}

	return plaintext, nil
}

// SetSecret encrypts and sets the API secret
func (ea *ExchangeAccount) SetSecret(plaintext string, encryptor Encryptor) error {
	if encryptor == nil {
		return nil
	}

	encrypted, err := encryptor.Encrypt(plaintext)
	if err != nil {
		return err
	}

	ea.SecretEncrypted = encrypted
	return nil
}

// GetSecret decrypts and returns the API secret
func (ea *ExchangeAccount) GetSecret(encryptor Encryptor) (string, error) {
	if encryptor == nil || len(ea.SecretEncrypted) == 0 {
		return "", nil
	}

	plaintext, err := encryptor.Decrypt(ea.SecretEncrypted)
	if err != nil {
		return "", err
	}

	return plaintext, nil
}

// SetPassphrase encrypts and sets the passphrase (for OKX)
func (ea *ExchangeAccount) SetPassphrase(plaintext string, encryptor Encryptor) error {
	if encryptor == nil || plaintext == "" {
		return nil
	}

	encrypted, err := encryptor.Encrypt(plaintext)
	if err != nil {
		return err
	}

	ea.Passphrase = encrypted
	return nil
}

// GetPassphrase decrypts and returns the passphrase
func (ea *ExchangeAccount) GetPassphrase(encryptor Encryptor) (string, error) {
	if encryptor == nil || len(ea.Passphrase) == 0 {
		return "", nil
	}

	plaintext, err := encryptor.Decrypt(ea.Passphrase)
	if err != nil {
		return "", err
	}

	return plaintext, nil
}
