package exchangefactory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/exchanges/binance"
	"prometheus/internal/adapters/exchanges/bybit"
	"prometheus/internal/adapters/exchanges/okx"
	"prometheus/internal/adapters/exchanges/ratelimit"
	"prometheus/internal/adapters/exchanges/retry"
	"prometheus/internal/domain/exchange_account"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// UserExchangeFactory creates exchange clients using user-specific API keys
// for trading operations. Each user gets their own client instance.
type UserExchangeFactory struct {
	encryptor *crypto.Encryptor
	clients   map[uuid.UUID]exchanges.Exchange // Keyed by exchange account ID
	limiters  *ratelimit.ExchangeLimiters
	retry     *retry.Middleware
	mu        sync.RWMutex
	log       *logger.Logger
}

// NewUserExchangeFactory creates a new user exchange factory
func NewUserExchangeFactory(encryptor *crypto.Encryptor) *UserExchangeFactory {
	return &UserExchangeFactory{
		encryptor: encryptor,
		clients:   make(map[uuid.UUID]exchanges.Exchange),
		limiters:  ratelimit.NewExchangeLimiters(),
		retry:     retry.New(retry.DefaultConfig()),
		log:       logger.Get().With("component", "user_exchange_factory"),
	}
}

// GetClient returns an exchange client for the given account
// Creates a new client if one doesn't exist, otherwise returns cached client
func (f *UserExchangeFactory) GetClient(ctx context.Context, account *exchange_account.ExchangeAccount) (exchanges.Exchange, error) {
	if account == nil {
		return nil, errors.Wrapf(errors.ErrInvalidInput, "exchange account is nil")
	}

	// Check cache first
	f.mu.RLock()
	if client, ok := f.clients[account.ID]; ok {
		f.mu.RUnlock()
		return client, nil
	}
	f.mu.RUnlock()

	// Create new client
	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring lock
	if client, ok := f.clients[account.ID]; ok {
		return client, nil
	}

	// Decrypt API keys
	apiKey, err := f.encryptor.Decrypt(account.APIKeyEncrypted)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decrypt API key")
	}

	secret, err := f.encryptor.Decrypt(account.SecretEncrypted)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decrypt secret")
	}

	var passphrase string
	if len(account.Passphrase) > 0 {
		passphrase, err = f.encryptor.Decrypt(account.Passphrase)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decrypt passphrase")
		}
	}

	// Create exchange-specific client
	var client exchanges.Exchange

	switch account.Exchange {
	case exchange_account.ExchangeBinance:
		client, err = binance.NewClient(binance.Config{
			APIKey:    apiKey,
			SecretKey: secret,
			Testnet:   account.IsTestnet,
		})

	case exchange_account.ExchangeBybit:
		client, err = bybit.NewClient(bybit.Config{
			APIKey:    apiKey,
			SecretKey: secret,
			Testnet:   account.IsTestnet,
		})

	case exchange_account.ExchangeOKX:
		client, err = okx.NewClient(okx.Config{
			APIKey:     apiKey,
			SecretKey:  secret,
			Passphrase: passphrase,
			Testnet:    account.IsTestnet,
		})

	default:
		return nil, errors.Wrapf(errors.ErrNotFound, "unsupported exchange: %s", account.Exchange)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create %s client", account.Exchange)
	}

	// Wrap with rate limiter and retry
	client = f.wrapClient(string(account.Exchange), client)

	// Cache client
	f.clients[account.ID] = client

	f.log.Infof("Created user exchange client for %s (account: %s, user: %s)",
		account.Exchange, account.ID, account.UserID)

	return client, nil
}

// GetClientByID returns an exchange client by account ID
// Requires the account to be already loaded
func (f *UserExchangeFactory) GetClientByID(accountID uuid.UUID) (exchanges.Exchange, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	client, ok := f.clients[accountID]
	return client, ok
}

// RemoveClient removes a client from the cache (useful when keys are rotated)
func (f *UserExchangeFactory) RemoveClient(accountID uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if client, ok := f.clients[accountID]; ok {
		if closer, ok := client.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				f.log.Errorf("failed to close client for account %s: %v", accountID, err)
			}
		}
		delete(f.clients, accountID)
		f.log.Infof("removed client for account %s", accountID)
	}
}

// wrapClient wraps the exchange client with rate limiting and retry logic
func (f *UserExchangeFactory) wrapClient(exchange string, client exchanges.Exchange) exchanges.Exchange {
	// This would ideally wrap the client with rate limiting and retry
	// For now, return the client as-is
	// TODO: Implement decorator pattern for rate limiting and retry
	return client
}

// GetLimiters returns the rate limiters for exchanges
func (f *UserExchangeFactory) GetLimiters() *ratelimit.ExchangeLimiters {
	return f.limiters
}

// Close closes all exchange clients
func (f *UserExchangeFactory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for accountID, client := range f.clients {
		if closer, ok := client.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				f.log.Errorf("failed to close client for account %s: %v", accountID, err)
			}
		}
	}

	f.clients = make(map[uuid.UUID]exchanges.Exchange)
	return nil
}
