package market_data

import (
	"context"
)

// Repository defines the interface for market data access (ClickHouse)
type Repository interface {
	// OHLCV operations
	InsertOHLCV(ctx context.Context, candles []OHLCV) error
	GetOHLCV(ctx context.Context, query OHLCVQuery) ([]OHLCV, error)
	GetLatestOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) ([]OHLCV, error)

	// Ticker operations
	InsertTicker(ctx context.Context, ticker *Ticker) error
	GetLatestTicker(ctx context.Context, exchange, symbol string) (*Ticker, error)

	// Order book operations
	InsertOrderBook(ctx context.Context, snapshot *OrderBookSnapshot) error
	GetLatestOrderBook(ctx context.Context, exchange, symbol string) (*OrderBookSnapshot, error)

	// Trades operations
	InsertTrades(ctx context.Context, trades []Trade) error
	GetRecentTrades(ctx context.Context, exchange, symbol string, limit int) ([]Trade, error)

	// Funding rate operations
	InsertFundingRate(ctx context.Context, fundingRate *FundingRate) error
	GetLatestFundingRate(ctx context.Context, exchange, symbol string) (*FundingRate, error)

	// Open interest operations
	InsertOpenInterest(ctx context.Context, oi *OpenInterest) error
	GetLatestOpenInterest(ctx context.Context, exchange, symbol string) (*OpenInterest, error)
}
