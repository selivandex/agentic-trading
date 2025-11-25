package exchanges

import (
	"fmt"
	"strings"
	"sync"

	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/exchanges/binance"
	"prometheus/internal/adapters/exchanges/bybit"
	"prometheus/internal/adapters/exchanges/okx"
	"prometheus/internal/domain/exchange_account"
	"prometheus/pkg/crypto"
)

// Factory constructs user-specific exchange clients with pooling.
type Factory interface {
	CreateClient(account *exchange_account.ExchangeAccount, encryptor *crypto.Encryptor) (Exchange, error)
	GetClient(exchangeType exchange_account.ExchangeType) (Exchange, error)
	ListExchanges() []exchange_account.ExchangeType
}

// FactoryOption customizes factory behavior.
type FactoryOption func(*factory)

// SystemCredential configures static shared clients.
type SystemCredential struct {
	Exchange   exchange_account.ExchangeType
	APIKey     string
	Secret     string
	Passphrase string
	Testnet    bool
	Market     MarketType
}

type factory struct {
	mu            sync.RWMutex
	userClients   map[string]Exchange
	sharedClients map[exchange_account.ExchangeType]Exchange
	systemCreds   map[exchange_account.ExchangeType]SystemCredential
}

// NewFactory builds a pooled exchange factory.
func NewFactory(opts ...FactoryOption) Factory {
	f := &factory{
		userClients:   make(map[string]Exchange),
		sharedClients: make(map[exchange_account.ExchangeType]Exchange),
		systemCreds:   make(map[exchange_account.ExchangeType]SystemCredential),
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithSystemCredentials registers shared credentials used by GetClient.
func WithSystemCredentials(creds []SystemCredential) FactoryOption {
	return func(f *factory) {
		for _, cred := range creds {
			f.systemCreds[cred.Exchange] = cred
		}
	}
}

func (f *factory) CreateClient(account *exchange_account.ExchangeAccount, encryptor *crypto.Encryptor) (Exchange, error) {
	if account == nil {
		return nil, fmt.Errorf("account is nil")
	}
	if encryptor == nil {
		return nil, fmt.Errorf("encryptor is nil")
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
		return nil, fmt.Errorf("decrypt api key: %w", err)
	}

	secret, err := encryptor.Decrypt(account.SecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret key: %w", err)
	}

	var passphrase string
	if len(account.Passphrase) > 0 {
		passphrase, err = encryptor.Decrypt(account.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("decrypt passphrase: %w", err)
		}
	}

	market := inferMarket(account.Permissions)

	client, err := instantiateClient(account.Exchange, adapterCredentials{
		APIKey:     apiKey,
		Secret:     secret,
		Passphrase: passphrase,
		Testnet:    account.IsTestnet,
		Market:     market,
	})
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.userClients[key] = client
	f.mu.Unlock()

	return client, nil
}

func (f *factory) GetClient(exchangeType exchange_account.ExchangeType) (Exchange, error) {
	f.mu.RLock()
	if client, ok := f.sharedClients[exchangeType]; ok {
		f.mu.RUnlock()
		return client, nil
	}
	f.mu.RUnlock()

	cred, ok := f.systemCreds[exchangeType]
	if !ok {
		return nil, fmt.Errorf("no shared credentials configured for %s", exchangeType)
	}

	client, err := instantiateClient(exchangeType, adapterCredentials{
		APIKey:     cred.APIKey,
		Secret:     cred.Secret,
		Passphrase: cred.Passphrase,
		Testnet:    cred.Testnet,
		Market:     cred.Market,
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

type adapterCredentials struct {
	APIKey     string
	Secret     string
	Passphrase string
	Testnet    bool
	Market     MarketType
}

func instantiateClient(exchangeType exchange_account.ExchangeType, cred adapterCredentials) (Exchange, error) {
	switch exchangeType {
	case exchange_account.ExchangeBinance:
		return binance.NewClient(binance.Config{
			APIKey:    cred.APIKey,
			SecretKey: cred.Secret,
			Market:    cred.Market,
			Testnet:   cred.Testnet,
		})
	case exchange_account.ExchangeBybit:
		return bybit.NewClient(bybit.Config{
			APIKey:    cred.APIKey,
			SecretKey: cred.Secret,
			Market:    cred.Market,
			Testnet:   cred.Testnet,
		})
	case exchange_account.ExchangeOKX:
		return okx.NewClient(okx.Config{
			APIKey:     cred.APIKey,
			SecretKey:  cred.Secret,
			Passphrase: cred.Passphrase,
			Market:     cred.Market,
			Testnet:    cred.Testnet,
		})
	default:
		return nil, fmt.Errorf("unsupported exchange: %s", exchangeType)
	}
}

func inferMarket(perms []string) MarketType {
	for _, perm := range perms {
		switch strings.ToLower(perm) {
		case "futures", "perp", "perpetual", "derivatives":
			return MarketTypeLinearPerp
		}
	}
	return MarketTypeSpot
}

// CentralFactory uses ENV-provided keys for market data collectors.
type CentralFactory interface {
	GetClient(exchange string) (Exchange, error)
	ListExchanges() []string
}

type centralFactory struct {
	cfg     config.MarketDataConfig
	mu      sync.RWMutex
	clients map[string]Exchange
}

// NewCentralFactory returns a factory backed by central API keys.
func NewCentralFactory(cfg config.MarketDataConfig) CentralFactory {
	return &centralFactory{
		cfg:     cfg,
		clients: make(map[string]Exchange),
	}
}

func (f *centralFactory) GetClient(exchange string) (Exchange, error) {
	f.mu.RLock()
	if client, ok := f.clients[exchange]; ok {
		f.mu.RUnlock()
		return client, nil
	}
	f.mu.RUnlock()

	var (
		client Exchange
		err    error
	)

	switch strings.ToLower(exchange) {
	case "binance":
		client, err = binance.NewClient(binance.Config{
			APIKey:  f.cfg.BinanceAPIKey,
			SecretKey: f.cfg.BinanceSecret,
			Market:  MarketTypeSpot,
		})
	case "bybit":
		client, err = bybit.NewClient(bybit.Config{
			APIKey:  f.cfg.BybitAPIKey,
			SecretKey: f.cfg.BybitSecret,
			Market:  MarketTypeLinearPerp,
		})
	case "okx":
		client, err = okx.NewClient(okx.Config{
			APIKey:     f.cfg.OKXAPIKey,
			SecretKey:  f.cfg.OKXSecret,
			Passphrase: f.cfg.OKXPassphrase,
			Market:     MarketTypeSpot,
		})
	default:
		return nil, fmt.Errorf("unsupported exchange: %s", exchange)
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

