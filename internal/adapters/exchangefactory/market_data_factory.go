package exchangefactory

import (
	"sync"

	"prometheus/internal/adapters/config"
	"prometheus/internal/adapters/exchanges"
	"prometheus/internal/adapters/exchanges/binance"
	"prometheus/internal/adapters/exchanges/bybit"
	"prometheus/internal/adapters/exchanges/okx"
	"prometheus/internal/adapters/exchanges/ratelimit"
	"prometheus/internal/adapters/exchanges/retry"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// MarketDataFactory creates exchange clients using centralized API keys
// for market data collection. These clients are shared across all workers.
type MarketDataFactory struct {
	cfg      config.MarketDataConfig
	clients  map[string]exchanges.Exchange
	limiters *ratelimit.ExchangeLimiters
	retry    *retry.Middleware
	mu       sync.RWMutex
	log      *logger.Logger
}

// NewMarketDataFactory creates a new market data factory
func NewMarketDataFactory(cfg config.MarketDataConfig) *MarketDataFactory {
	return &MarketDataFactory{
		cfg:      cfg,
		clients:  make(map[string]exchanges.Exchange),
		limiters: ratelimit.NewExchangeLimiters(),
		retry:    retry.New(retry.DefaultConfig()),
		log:      logger.Get().With("component", "market_data_factory"),
	}
}

// GetClient returns an exchange client for market data collection
// Uses centralized API keys from configuration
func (f *MarketDataFactory) GetClient(exchange string) (exchanges.Exchange, error) {
	// Check cache first
	f.mu.RLock()
	if client, ok := f.clients[exchange]; ok {
		f.mu.RUnlock()
		return client, nil
	}
	f.mu.RUnlock()

	// Create new client
	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring lock
	if client, ok := f.clients[exchange]; ok {
		return client, nil
	}

	var client exchanges.Exchange
	var err error

	switch exchange {
	case "binance":
		client, err = f.createBinanceClient()
	case "bybit":
		client, err = f.createBybitClient()
	case "okx":
		client, err = f.createOKXClient()
	default:
		return nil, errors.Wrapf(errors.ErrNotFound, "unsupported exchange: %s", exchange)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create %s client", exchange)
	}

	// Wrap with rate limiter and retry
	client = f.wrapClient(exchange, client)

	// Cache client
	f.clients[exchange] = client

	f.log.Infof("Created market data client for %s", exchange)

	return client, nil
}

func (f *MarketDataFactory) createBinanceClient() (exchanges.Exchange, error) {
	if f.cfg.Binance.APIKey == "" {
		f.log.Warn("Binance API key not configured, using public endpoints only")
	}

	return binance.NewClient(binance.Config{
		APIKey:    f.cfg.Binance.APIKey,
		SecretKey: f.cfg.Binance.Secret,
		Testnet:   false,
	})
}

func (f *MarketDataFactory) createBybitClient() (exchanges.Exchange, error) {
	if f.cfg.Bybit.APIKey == "" {
		f.log.Warn("Bybit API key not configured, using public endpoints only")
	}

	return bybit.NewClient(bybit.Config{
		APIKey:    f.cfg.Bybit.APIKey,
		SecretKey: f.cfg.Bybit.Secret,
		Testnet:   false,
	})
}

func (f *MarketDataFactory) createOKXClient() (exchanges.Exchange, error) {
	if f.cfg.OKX.APIKey == "" {
		f.log.Warn("OKX API key not configured, using public endpoints only")
	}

	return okx.NewClient(okx.Config{
		APIKey:     f.cfg.OKX.APIKey,
		SecretKey:  f.cfg.OKX.Secret,
		Passphrase: f.cfg.OKX.Passphrase,
		Testnet:    false,
	})
}

// wrapClient wraps the exchange client with rate limiting and retry logic
func (f *MarketDataFactory) wrapClient(exchange string, client exchanges.Exchange) exchanges.Exchange {
	// This would ideally wrap the client with rate limiting and retry
	// For now, return the client as-is
	// TODO: Implement decorator pattern for rate limiting and retry
	return client
}

// ListExchanges returns the list of supported exchanges
func (f *MarketDataFactory) ListExchanges() []string {
	return []string{"binance", "bybit", "okx"}
}

// GetLimiters returns the rate limiters for exchanges
func (f *MarketDataFactory) GetLimiters() *ratelimit.ExchangeLimiters {
	return f.limiters
}

// Close closes all exchange clients
func (f *MarketDataFactory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for exchange, client := range f.clients {
		if closer, ok := client.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				f.log.Errorf("Failed to close %s client: %v", exchange, err)
			}
		}
	}

	f.clients = make(map[string]exchanges.Exchange)
	return nil
}
