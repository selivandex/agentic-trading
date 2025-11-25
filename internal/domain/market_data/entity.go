package market_data

import "time"

// OHLCV represents candlestick data
type OHLCV struct {
	Exchange    string    `ch:"exchange"`
	Symbol      string    `ch:"symbol"`
	Timeframe   string    `ch:"timeframe"` // 1m, 5m, 15m, 1h, 4h, 1d
	OpenTime    time.Time `ch:"open_time"`
	Open        float64   `ch:"open"`
	High        float64   `ch:"high"`
	Low         float64   `ch:"low"`
	Close       float64   `ch:"close"`
	Volume      float64   `ch:"volume"`
	QuoteVolume float64   `ch:"quote_volume"`
	Trades      int64     `ch:"trades"`
}

// Ticker represents real-time ticker data
type Ticker struct {
	Exchange     string    `ch:"exchange"`
	Symbol       string    `ch:"symbol"`
	Timestamp    time.Time `ch:"timestamp"`
	Price        float64   `ch:"price"`
	Bid          float64   `ch:"bid"`
	Ask          float64   `ch:"ask"`
	Volume24h    float64   `ch:"volume_24h"`
	Change24h    float64   `ch:"change_24h"`
	High24h      float64   `ch:"high_24h"`
	Low24h       float64   `ch:"low_24h"`
	FundingRate  float64   `ch:"funding_rate"`  // Futures
	OpenInterest float64   `ch:"open_interest"` // Futures
}

// OrderBookSnapshot represents order book state at a point in time
type OrderBookSnapshot struct {
	Exchange  string    `ch:"exchange"`
	Symbol    string    `ch:"symbol"`
	Timestamp time.Time `ch:"timestamp"`
	Bids      string    `ch:"bids"`      // JSON array
	Asks      string    `ch:"asks"`      // JSON array
	BidDepth  float64   `ch:"bid_depth"` // Total bid volume
	AskDepth  float64   `ch:"ask_depth"` // Total ask volume
}

// Trade represents a single trade (tape)
type Trade struct {
	Exchange  string    `ch:"exchange"`
	Symbol    string    `ch:"symbol"`
	Timestamp time.Time `ch:"timestamp"`
	TradeID   string    `ch:"trade_id"`
	Price     float64   `ch:"price"`
	Quantity  float64   `ch:"quantity"`
	Side      string    `ch:"side"` // buy, sell
	IsBuyer   bool      `ch:"is_buyer"`
}

// OHLCVQuery represents query parameters for OHLCV data
type OHLCVQuery struct {
	Exchange  string
	Symbol    string
	Timeframe string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

