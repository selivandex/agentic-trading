package exchangefactory

import (
	"fmt"
	"strings"
	"sync"

	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/exchanges/binance"
	"prometheus/internal/adapters/exchanges/bybit"
	"prometheus/internal/adapters/exchanges/okx"
	"prometheus/internal/domain/exchange_account"
	"prometheus/pkg/crypto"
	"prometheus/pkg/errors"
)

// Option customizes factory behavior.
type Option func(*factory)

// WithSystemCredentials configures shared exchange clients.
func WithSystemCredentials(creds []exchanges.SystemCredential) Option {
	return func(f *factory) {
		for _, cred := range creds {
			f.systemCreds[cred.Exchange] = cred
		}
	}
}

// NewFactory creates a pooled exchange factory implementation.
func NewFactory(opts ...Option) exchanges.Factory {
	f := &factory{
		userClients:   make(map[string]exchanges.Exchange),
		sharedClients: make(map[exchange_account.ExchangeType]exchanges.Exchange),
		systemCreds:   make(map[exchange_account.ExchangeType]exchanges.SystemCredential),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

type factory struct {
	mu            sync.RWMutex
	userClients   map[string]exchanges.Exchange
	sharedClients map[exchange_account.ExchangeType]exchanges.Exchange
	systemCreds   map[exchange_account.ExchangeType]exchanges.SystemCredential
}

func (f *factory) CreateClient(account *exchange_account.ExchangeAccount, encryptor *crypto.Encryptor) (exchanges.Exchange, error) {
	if account == nil {
		return nil, errors.ErrInvalidInput
	}
	if encryptor == nil {
		return nil, errors.ErrInvalidInput
	}

	key := fmt.Sprintf("%s:%s", account.Exchange, account.ID)

	f.mu.RLock()
	if client, ok := f.userClients[key]; ok {
		f.mu.RUnlock()
		return client, nil
	}
	f.mu.RUnlock()

	apiKey, err := encryptor.Decrypt(account.APIKeyEncrypted)
	if err != nil {
		return nil, errors.Wrap(err, "decrypt api key")
	}
	secret, err := encryptor.Decrypt(account.SecretEncrypted)
	if err != nil {
		return nil, errors.Wrap(err, "decrypt secret")
	}

	var passphrase string
	if len(account.Passphrase) > 0 {
		passphrase, err = encryptor.Decrypt(account.Passphrase)
		if err != nil {
			return nil, errors.Wrap(err, "decrypt passphrase")
		}
	}

	client, err := instantiateClient(account.Exchange, credentialBundle{
		apiKey:     apiKey,
		secret:     secret,
		passphrase: passphrase,
		testnet:    account.IsTestnet,
		market:     inferMarket(account.Permissions),
	})
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.userClients[key] = client
	f.mu.Unlock()

	return client, nil
}

func (f *factory) GetClient(exchangeType exchange_account.ExchangeType) (exchanges.Exchange, error) {
	f.mu.RLock()
	if client, ok := f.sharedClients[exchangeType]; ok {
		f.mu.RUnlock()
		return client, nil
	}
	f.mu.RUnlock()

	cred, ok := f.systemCreds[exchangeType]
	if !ok {
		return nil, errors.Wrapf(errors.ErrNotFound, "no shared credentials configured for %s", exchangeType)
	}

	client, err := instantiateClient(exchangeType, credentialBundle{
		apiKey:     cred.APIKey,
		secret:     cred.Secret,
		passphrase: cred.Passphrase,
		testnet:    cred.Testnet,
		market:     cred.Market,
	})
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.sharedClients[exchangeType] = client
	f.mu.Unlock()

	return client, nil
}

func (f *factory) ListExchanges() []exchange_account.ExchangeType {
	return []exchange_account.ExchangeType{
		exchange_account.ExchangeBinance,
		exchange_account.ExchangeBybit,
		exchange_account.ExchangeOKX,
	}
}

type credentialBundle struct {
	apiKey     string
	secret     string
	passphrase string
	testnet    bool
	market     exchanges.MarketType
}

func instantiateClient(exchangeType exchange_account.ExchangeType, cred credentialBundle) (exchanges.Exchange, error) {
	switch exchangeType {
	case exchange_account.ExchangeBinance:
		return binance.NewClient(binance.Config{
			APIKey:    cred.apiKey,
			SecretKey: cred.secret,
			Market:    cred.market,
			Testnet:   cred.testnet,
		})
	case exchange_account.ExchangeBybit:
		return bybit.NewClient(bybit.Config{
			APIKey:    cred.apiKey,
			SecretKey: cred.secret,
			Market:    cred.market,
			Testnet:   cred.testnet,
		})
	case exchange_account.ExchangeOKX:
		return okx.NewClient(okx.Config{
			APIKey:     cred.apiKey,
			SecretKey:  cred.secret,
			Passphrase: cred.passphrase,
			Market:     cred.market,
			Testnet:    cred.testnet,
		})
	default:
		return nil, errors.Wrapf(errors.ErrInvalidInput, "unsupported exchange: %s", exchangeType)
	}
}

func inferMarket(perms []string) exchanges.MarketType {
	for _, perm := range perms {
		switch strings.ToLower(perm) {
		case "futures", "perp", "perpetual", "derivatives":
			return exchanges.MarketTypeLinearPerp
		}
	}
	return exchanges.MarketTypeSpot
}

// Central factory implementation --------------------------------------------

type centralFactory struct {
	cfg     config.MarketDataConfig
	mu      sync.RWMutex
	clients map[string]exchanges.Exchange
}

// NewCentralFactory returns a factory backed by ENV API keys.
func NewCentralFactory(cfg config.MarketDataConfig) exchanges.CentralFactory {
	return &centralFactory{
		cfg:     cfg,
		clients: make(map[string]exchanges.Exchange),
	}
}

func (f *centralFactory) GetClient(exchange string) (exchanges.Exchange, error) {
	f.mu.RLock()
	if client, ok := f.clients[exchange]; ok {
		f.mu.RUnlock()
		return client, nil
	}
	f.mu.RUnlock()

	var (
		client exchanges.Exchange
		err    error
	)

	switch strings.ToLower(exchange) {
	case "binance":
		client, err = binance.NewClient(binance.Config{
			APIKey:    f.cfg.Binance.APIKey,
			SecretKey: f.cfg.Binance.Secret,
			Market:    exchanges.MarketTypeSpot,
		})
	case "bybit":
		client, err = bybit.NewClient(bybit.Config{
			APIKey:    f.cfg.Bybit.APIKey,
			SecretKey: f.cfg.Bybit.Secret,
			Market:    exchanges.MarketTypeLinearPerp,
		})
	case "okx":
		client, err = okx.NewClient(okx.Config{
			APIKey:     f.cfg.OKX.APIKey,
			SecretKey:  f.cfg.OKX.Secret,
			Passphrase: f.cfg.OKX.Passphrase,
			Market:     exchanges.MarketTypeSpot,
		})
	default:
		return nil, errors.Wrapf(errors.ErrInvalidInput, "unsupported exchange: %s", exchange)
	}

	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.clients[exchange] = client
	f.mu.Unlock()

	return client, nil
}

func (f *centralFactory) ListExchanges() []string {
	return []string{"binance", "bybit", "okx"}
}
