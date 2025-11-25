package exchanges

import (
	"context"
)

// Exchange defines the unified contract each exchange adapter must satisfy.
type Exchange interface {
	Name() string

	// Market data
	GetTicker(ctx context.Context, symbol string) (*Ticker, error)
	GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBook, error)
	GetOHLCV(ctx context.Context, symbol string, timeframe string, limit int) ([]OHLCV, error)
	GetTrades(ctx context.Context, symbol string, limit int) ([]Trade, error)
	GetFundingRate(ctx context.Context, symbol string) (*FundingRate, error)
	GetOpenInterest(ctx context.Context, symbol string) (*OpenInterest, error)

	// Account
	GetBalance(ctx context.Context) (*Balance, error)
	GetPositions(ctx context.Context) ([]Position, error)
	GetOpenOrders(ctx context.Context, symbol string) ([]Order, error)

	// Trading
	PlaceOrder(ctx context.Context, req *OrderRequest) (*Order, error)
	CancelOrder(ctx context.Context, symbol, orderID string) error

	// Futures specific
	SetLeverage(ctx context.Context, symbol string, leverage int) error
	SetMarginMode(ctx context.Context, symbol string, mode MarginMode) error
}

// SpotExchange marks an exchange that only supports spot operations.
type SpotExchange interface {
	Exchange
}

// FuturesExchange marks an exchange that supports futures specific operations.
type FuturesExchange interface {
	Exchange
}

