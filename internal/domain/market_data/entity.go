package market_data

import "time"

// OHLCV represents candlestick data
type OHLCV struct {
	Exchange            string    `ch:"exchange"`
	Symbol              string    `ch:"symbol"`
	Timeframe           string    `ch:"timeframe"`   // 1m, 5m, 15m, 1h, 4h, 1d
	MarketType          string    `ch:"market_type"` // spot, linear_perp, inverse_perp
	OpenTime            time.Time `ch:"open_time"`
	CloseTime           time.Time `ch:"close_time"`
	Open                float64   `ch:"open"`
	High                float64   `ch:"high"`
	Low                 float64   `ch:"low"`
	Close               float64   `ch:"close"`
	Volume              float64   `ch:"volume"`
	QuoteVolume         float64   `ch:"quote_volume"`
	Trades              uint64    `ch:"trades"`
	TakerBuyBaseVolume  float64   `ch:"taker_buy_base_volume"`  // Buying pressure indicator
	TakerBuyQuoteVolume float64   `ch:"taker_buy_quote_volume"` // Quote volume from buy orders
	IsClosed            bool      `ch:"is_closed"`              // Whether kline is closed (final)
	EventTime           time.Time `ch:"event_time"`             // Exchange event timestamp (used for versioning)
}

// MarkPrice represents mark price and funding rate for derivatives
// ONLY for futures/perpetuals, updated every 1-3 seconds
type MarkPrice struct {
	Exchange             string    `ch:"exchange"`
	Symbol               string    `ch:"symbol"`
	MarketType           string    `ch:"market_type"` // futures, linear_perp, inverse_perp
	Timestamp            time.Time `ch:"timestamp"`
	MarkPrice            float64   `ch:"mark_price"`
	IndexPrice           float64   `ch:"index_price"`
	EstimatedSettlePrice float64   `ch:"estimated_settle_price"`
	FundingRate          float64   `ch:"funding_rate"`
	NextFundingTime      time.Time `ch:"next_funding_time"`
	EventTime            time.Time `ch:"event_time"`
}

// Ticker represents 24hr rolling window statistics
// For both spot and futures markets
type Ticker struct {
	Exchange           string    `ch:"exchange"`
	Symbol             string    `ch:"symbol"`
	MarketType         string    `ch:"market_type"` // spot or futures
	Timestamp          time.Time `ch:"timestamp"`
	LastPrice          float64   `ch:"last_price"`
	OpenPrice          float64   `ch:"open_price"`
	HighPrice          float64   `ch:"high_price"`
	LowPrice           float64   `ch:"low_price"`
	Volume             float64   `ch:"volume"`
	QuoteVolume        float64   `ch:"quote_volume"`
	PriceChange        float64   `ch:"price_change"`
	PriceChangePercent float64   `ch:"price_change_percent"`
	WeightedAvgPrice   float64   `ch:"weighted_avg_price"`
	TradeCount         uint64    `ch:"trade_count"`
	EventTime          time.Time `ch:"event_time"`
	CollectedAt        time.Time `ch:"collected_at"` // When we collected this data
}

// OrderBookSnapshot represents order book state at a point in time
type OrderBookSnapshot struct {
	Exchange    string    `ch:"exchange"`
	Symbol      string    `ch:"symbol"`
	MarketType  string    `ch:"market_type"` // spot or futures
	Timestamp   time.Time `ch:"timestamp"`
	Bids        string    `ch:"bids"`      // JSON array
	Asks        string    `ch:"asks"`      // JSON array
	BidDepth    float64   `ch:"bid_depth"` // Total bid volume
	AskDepth    float64   `ch:"ask_depth"` // Total ask volume
	EventTime   time.Time `ch:"event_time"`
	CollectedAt time.Time `ch:"collected_at"` // When we collected this data
}

// Trade represents aggregated trade (tape/time & sales)
// Aggregates trades that fill from the same taker order
type Trade struct {
	Exchange     string    `ch:"exchange"`
	Symbol       string    `ch:"symbol"`
	MarketType   string    `ch:"market_type"` // spot or futures
	Timestamp    time.Time `ch:"timestamp"`
	TradeID      int64     `ch:"trade_id"`
	AggTradeID   int64     `ch:"agg_trade_id"`
	Price        float64   `ch:"price"`
	Quantity     float64   `ch:"quantity"`
	FirstTradeID int64     `ch:"first_trade_id"`
	LastTradeID  int64     `ch:"last_trade_id"`
	IsBuyerMaker bool      `ch:"is_buyer_maker"` // true if seller was the maker
	EventTime    time.Time `ch:"event_time"`
	CollectedAt  time.Time `ch:"collected_at"` // When we collected this data
}

// Side returns the side of the trade ("buy" or "sell" from taker perspective)
func (t *Trade) Side() string {
	if t.IsBuyerMaker {
		return "sell" // Taker was seller (aggressor sell)
	}
	return "buy" // Taker was buyer (aggressor buy)
}

// Liquidation represents a force liquidation order
// ONLY for futures/perpetuals
type Liquidation struct {
	Exchange   string    `ch:"exchange"`
	Symbol     string    `ch:"symbol"`
	MarketType string    `ch:"market_type"` // futures, linear_perp, inverse_perp
	Timestamp  time.Time `ch:"timestamp"`
	Side       string    `ch:"side"`       // LONG or SHORT position being liquidated
	OrderType  string    `ch:"order_type"` // LIMIT, MARKET
	Price      float64   `ch:"price"`
	Quantity   float64   `ch:"quantity"`
	Value      float64   `ch:"value"` // price * quantity in USD
	EventTime  time.Time `ch:"event_time"`
}

// OpenInterest represents open interest for futures contracts
// Note: On Binance, only available via REST API (updates every ~5min)
type OpenInterest struct {
	Exchange  string    `ch:"exchange"`
	Symbol    string    `ch:"symbol"`
	Timestamp time.Time `ch:"timestamp"`
	Amount    float64   `ch:"amount"` // Open interest in contracts or USD
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
