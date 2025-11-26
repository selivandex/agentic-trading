package exchanges

import (
	"context"
	"prometheus/internal/domain/exchange_account"
	"prometheus/pkg/crypto"
)

// SystemCredential holds credentials for system-wide market data collection
type SystemCredential struct {
	Exchange   exchange_account.ExchangeType
	APIKey     string
	Secret     string
	Passphrase string
	Testnet    bool
	Market     MarketType
}

// Factory creates exchange clients for user accounts
type Factory interface {
	CreateClient(account *exchange_account.ExchangeAccount, encryptor *crypto.Encryptor) (Exchange, error)
	GetClient(exchangeType exchange_account.ExchangeType) (Exchange, error)
	ListExchanges() []exchange_account.ExchangeType
}

// CentralFactory creates exchange clients for centralized market data collection
type CentralFactory interface {
	GetClient(exchange string) (Exchange, error)
	ListExchanges() []string
}

// WebSocketClient interface for real-time WebSocket connections
type WebSocketClient interface {
	Connect(ctx context.Context) error
	Disconnect() error
	SubscribeOrderBook(symbol string, callback func(*OrderBook)) error
	SubscribeTrades(symbol string, callback func(*Trade)) error
	SubscribeTicker(symbol string, callback func(*Ticker)) error
	IsConnected() bool
	Ping() error
}
