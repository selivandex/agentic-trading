package testsupport

import (
	"context"
	"fmt"
	"testing"
	"time"

	"prometheus/internal/adapters/clickhouse"
	"prometheus/internal/adapters/config"
	"prometheus/internal/domain/market_data"
)

// ClickHouseTestHelper manages cleanup for ClickHouse integration tests.
type ClickHouseTestHelper struct {
	client *clickhouse.Client
}

// NewClickHouseTestHelper creates a ClickHouse client for tests.
func NewClickHouseTestHelper(t *testing.T, cfg config.ClickHouseConfig) *ClickHouseTestHelper {
	t.Helper()

	client, err := clickhouse.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to connect to clickhouse: %v", err)
	}

	helper := &ClickHouseTestHelper{client: client}
	t.Cleanup(func() { _ = client.Close() })
	return helper
}

// CreateTempTable creates a temporary table and registers cleanup.
func (h *ClickHouseTestHelper) CreateTempTable(t *testing.T, schema string) string {
	t.Helper()

	table := fmt.Sprintf("tmp_test_%d", time.Now().UnixNano())
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s) ENGINE = MergeTree() ORDER BY tuple()", table, schema)

	if err := h.client.Exec(context.Background(), query); err != nil {
		t.Fatalf("failed to create clickhouse table: %v", err)
	}

	t.Cleanup(func() {
		_ = h.client.Exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	})

	return table
}

// CleanupTable drops the provided table immediately.
func (h *ClickHouseTestHelper) CleanupTable(ctx context.Context, table string) error {
	return h.client.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
}

// TruncateTable removes all data from the table but keeps the structure
func (h *ClickHouseTestHelper) TruncateTable(ctx context.Context, table string) error {
	return h.client.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE IF EXISTS %s", table))
}

// CleanupTableData deletes data matching a filter condition
// Example: CleanupTableData(ctx, "ohlcv", "exchange = 'test_exchange'")
func (h *ClickHouseTestHelper) CleanupTableData(ctx context.Context, table, condition string) error {
	query := fmt.Sprintf("ALTER TABLE %s DELETE WHERE %s", table, condition)
	return h.client.Exec(ctx, query)
}

// RegisterTableCleanup schedules cleanup of specific table data after test completes
// This is useful when working with shared tables that shouldn't be dropped
func (h *ClickHouseTestHelper) RegisterTableCleanup(t *testing.T, table, condition string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Use DELETE for immediate cleanup (ALTER TABLE DELETE is async)
		query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, condition)
		_ = h.client.Exec(ctx, query)
	})
}

// CreateBatch is a generic function to insert test data into ClickHouse tables
// Usage: testsupport.CreateBatch(t, helper, testsupport.InsertOHLCV, candles)
func CreateBatch[T any](t *testing.T, helper *ClickHouseTestHelper, insertQuery string, items []T) {
	t.Helper()

	if len(items) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	batch, err := helper.client.Conn().PrepareBatch(ctx, insertQuery)
	if err != nil {
		t.Fatalf("failed to prepare batch: %v", err)
	}

	for _, item := range items {
		if err := batch.AppendStruct(&item); err != nil {
			t.Fatalf("failed to append item to batch: %v", err)
		}
	}

	if err := batch.Send(); err != nil {
		t.Fatalf("failed to send batch: %v", err)
	}
}

// Predefined insert queries for common tables
const (
	InsertOHLCV = `
		INSERT INTO ohlcv (
			exchange, symbol, timeframe, market_type, open_time, close_time,
			open, high, low, close, volume, quote_volume, trades,
			taker_buy_base_volume, taker_buy_quote_volume
		)
	`

	InsertTickers = `
		INSERT INTO tickers (
			exchange, symbol, timestamp, price, bid, ask,
			volume_24h, change_24h, high_24h, low_24h,
			funding_rate, open_interest
		)
	`

	InsertTrades = `
		INSERT INTO trades (
			exchange, symbol, timestamp, trade_id, price, quantity, side, is_buyer
		)
	`

	InsertFundingRates = `
		INSERT INTO funding_rates (
			exchange, symbol, timestamp, funding_rate, next_funding_time, mark_price, index_price
		)
	`
)

// Client exposes the raw ClickHouse client for queries.
func (h *ClickHouseTestHelper) Client() *clickhouse.Client {
	return h.client
}

// ========================================
// Fixture Builders for ClickHouse Tests
// ========================================

// OHLCVFixture provides builder pattern for creating test OHLCV candles
type OHLCVFixture struct {
	candle market_data.OHLCV
}

// NewOHLCVFixture creates a default OHLCV candle for testing
// Default: BTC/USDT on Binance, spot market, with realistic values
func NewOHLCVFixture() *OHLCVFixture {
	now := time.Now().Truncate(time.Minute)
	return &OHLCVFixture{
		candle: market_data.OHLCV{
			Exchange:            "binance",
			Symbol:              "BTC/USDT",
			Timeframe:           "1h",
			MarketType:          "spot",
			OpenTime:            now,
			CloseTime:           now.Add(1 * time.Hour),
			Open:                50000.0,
			High:                51000.0,
			Low:                 49500.0,
			Close:               50500.0,
			Volume:              100.0,
			QuoteVolume:         5000000.0,
			Trades:              1000,
			TakerBuyBaseVolume:  55.0, // 55% buy pressure
			TakerBuyQuoteVolume: 2750000.0,
			IsClosed:            true,
			EventTime:           now,
		},
	}
}

// WithExchange sets the exchange
func (f *OHLCVFixture) WithExchange(exchange string) *OHLCVFixture {
	f.candle.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *OHLCVFixture) WithSymbol(symbol string) *OHLCVFixture {
	f.candle.Symbol = symbol
	return f
}

// WithTimeframe sets the timeframe
func (f *OHLCVFixture) WithTimeframe(timeframe string) *OHLCVFixture {
	f.candle.Timeframe = timeframe
	return f
}

// WithMarketType sets the market type
func (f *OHLCVFixture) WithMarketType(marketType string) *OHLCVFixture {
	f.candle.MarketType = marketType
	return f
}

// WithOpenTime sets the open time and calculates close time based on timeframe
func (f *OHLCVFixture) WithOpenTime(t time.Time) *OHLCVFixture {
	f.candle.OpenTime = t
	f.candle.CloseTime = t.Add(parseTimeframeDuration(f.candle.Timeframe))
	return f
}

// WithTimes sets both open and close times explicitly
func (f *OHLCVFixture) WithTimes(openTime, closeTime time.Time) *OHLCVFixture {
	f.candle.OpenTime = openTime
	f.candle.CloseTime = closeTime
	return f
}

// WithPrices sets OHLC prices
func (f *OHLCVFixture) WithPrices(open, high, low, close float64) *OHLCVFixture {
	f.candle.Open = open
	f.candle.High = high
	f.candle.Low = low
	f.candle.Close = close
	return f
}

// WithVolume sets base and quote volumes
func (f *OHLCVFixture) WithVolume(baseVolume, quoteVolume float64) *OHLCVFixture {
	f.candle.Volume = baseVolume
	f.candle.QuoteVolume = quoteVolume
	return f
}

// WithTrades sets the number of trades
func (f *OHLCVFixture) WithTrades(trades uint64) *OHLCVFixture {
	f.candle.Trades = trades
	return f
}

// WithBuyPressure sets taker buy volumes to achieve specific buy pressure percentage
// buyPressurePct should be between 0 and 100 (e.g., 60 for 60% buy pressure)
func (f *OHLCVFixture) WithBuyPressure(buyPressurePct float64) *OHLCVFixture {
	f.candle.TakerBuyBaseVolume = f.candle.Volume * (buyPressurePct / 100.0)
	f.candle.TakerBuyQuoteVolume = f.candle.QuoteVolume * (buyPressurePct / 100.0)
	return f
}

// WithTakerBuyVolumes sets taker buy volumes explicitly
func (f *OHLCVFixture) WithTakerBuyVolumes(baseVolume, quoteVolume float64) *OHLCVFixture {
	f.candle.TakerBuyBaseVolume = baseVolume
	f.candle.TakerBuyQuoteVolume = quoteVolume
	return f
}

// WithIsClosed sets whether the candle is closed/final
func (f *OHLCVFixture) WithIsClosed(isClosed bool) *OHLCVFixture {
	f.candle.IsClosed = isClosed
	return f
}

// WithEventTime sets the exchange event timestamp
func (f *OHLCVFixture) WithEventTime(t time.Time) *OHLCVFixture {
	f.candle.EventTime = t
	return f
}

// Bullish creates a bullish candle (close > open, high buy pressure)
func (f *OHLCVFixture) Bullish() *OHLCVFixture {
	basePrice := f.candle.Open
	f.candle.High = basePrice * 1.03
	f.candle.Low = basePrice * 0.99
	f.candle.Close = basePrice * 1.02
	return f.WithBuyPressure(65.0) // 65% buy pressure
}

// Bearish creates a bearish candle (close < open, low buy pressure)
func (f *OHLCVFixture) Bearish() *OHLCVFixture {
	basePrice := f.candle.Open
	f.candle.High = basePrice * 1.01
	f.candle.Low = basePrice * 0.97
	f.candle.Close = basePrice * 0.98
	return f.WithBuyPressure(35.0) // 35% buy pressure
}

// Build returns the constructed OHLCV candle
func (f *OHLCVFixture) Build() market_data.OHLCV {
	return f.candle
}

// BuildMany creates multiple candles with sequential timestamps
func (f *OHLCVFixture) BuildMany(count int) []market_data.OHLCV {
	candles := make([]market_data.OHLCV, count)
	duration := parseTimeframeDuration(f.candle.Timeframe)

	for i := 0; i < count; i++ {
		candle := f.candle
		candle.OpenTime = f.candle.OpenTime.Add(time.Duration(i) * duration)
		candle.CloseTime = candle.OpenTime.Add(duration)
		candle.EventTime = f.candle.EventTime.Add(time.Duration(i) * duration)
		candles[i] = candle
	}

	return candles
}

// TickerFixture provides builder pattern for creating test tickers
type TickerFixture struct {
	ticker market_data.Ticker
}

// NewTickerFixture creates a default ticker for testing
func NewTickerFixture() *TickerFixture {
	return &TickerFixture{
		ticker: market_data.Ticker{
			Exchange:     "binance",
			Symbol:       "BTC/USDT",
			Timestamp:    time.Now().Truncate(time.Second),
			Price:        50000.0,
			Bid:          49995.0,
			Ask:          50005.0,
			Volume24h:    10000.0,
			Change24h:    2.5,
			High24h:      51000.0,
			Low24h:       49000.0,
			FundingRate:  0.0001,
			OpenInterest: 1000000.0,
		},
	}
}

// WithExchange sets the exchange
func (f *TickerFixture) WithExchange(exchange string) *TickerFixture {
	f.ticker.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *TickerFixture) WithSymbol(symbol string) *TickerFixture {
	f.ticker.Symbol = symbol
	return f
}

// WithTimestamp sets the timestamp
func (f *TickerFixture) WithTimestamp(t time.Time) *TickerFixture {
	f.ticker.Timestamp = t
	return f
}

// WithPrice sets the current price and adjusts bid/ask around it
func (f *TickerFixture) WithPrice(price float64) *TickerFixture {
	f.ticker.Price = price
	f.ticker.Bid = price * 0.9999
	f.ticker.Ask = price * 1.0001
	return f
}

// WithPrices sets price, bid, and ask explicitly
func (f *TickerFixture) WithPrices(price, bid, ask float64) *TickerFixture {
	f.ticker.Price = price
	f.ticker.Bid = bid
	f.ticker.Ask = ask
	return f
}

// WithFundingRate sets the funding rate
func (f *TickerFixture) WithFundingRate(rate float64) *TickerFixture {
	f.ticker.FundingRate = rate
	return f
}

// Build returns the constructed ticker
func (f *TickerFixture) Build() market_data.Ticker {
	return f.ticker
}

// TradeFixture provides builder pattern for creating test trades
type TradeFixture struct {
	trade market_data.Trade
}

// NewTradeFixture creates a default trade for testing
func NewTradeFixture() *TradeFixture {
	return &TradeFixture{
		trade: market_data.Trade{
			Exchange:  "binance",
			Symbol:    "BTC/USDT",
			Timestamp: time.Now().Truncate(time.Millisecond),
			TradeID:   "test_trade_1",
			Price:     50000.0,
			Quantity:  0.1,
			Side:      "buy",
			IsBuyer:   true,
		},
	}
}

// WithExchange sets the exchange
func (f *TradeFixture) WithExchange(exchange string) *TradeFixture {
	f.trade.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *TradeFixture) WithSymbol(symbol string) *TradeFixture {
	f.trade.Symbol = symbol
	return f
}

// WithTimestamp sets the timestamp
func (f *TradeFixture) WithTimestamp(t time.Time) *TradeFixture {
	f.trade.Timestamp = t
	return f
}

// WithTradeID sets the trade ID
func (f *TradeFixture) WithTradeID(id string) *TradeFixture {
	f.trade.TradeID = id
	return f
}

// WithPrice sets the price
func (f *TradeFixture) WithPrice(price float64) *TradeFixture {
	f.trade.Price = price
	return f
}

// WithQuantity sets the quantity
func (f *TradeFixture) WithQuantity(qty float64) *TradeFixture {
	f.trade.Quantity = qty
	return f
}

// AsBuy makes this a buy trade
func (f *TradeFixture) AsBuy() *TradeFixture {
	f.trade.Side = "buy"
	f.trade.IsBuyer = true
	return f
}

// AsSell makes this a sell trade
func (f *TradeFixture) AsSell() *TradeFixture {
	f.trade.Side = "sell"
	f.trade.IsBuyer = false
	return f
}

// Build returns the constructed trade
func (f *TradeFixture) Build() market_data.Trade {
	return f.trade
}

// BuildMany creates multiple trades with sequential timestamps and IDs
func (f *TradeFixture) BuildMany(count int) []market_data.Trade {
	trades := make([]market_data.Trade, count)

	for i := 0; i < count; i++ {
		trade := f.trade
		trade.Timestamp = f.trade.Timestamp.Add(time.Duration(i) * time.Millisecond)
		trade.TradeID = f.trade.TradeID + "_" + string(rune('0'+i))
		trades[i] = trade
	}

	return trades
}

// FundingRateFixture provides builder pattern for creating test funding rates
type FundingRateFixture struct {
	fundingRate market_data.FundingRate
}

// NewFundingRateFixture creates a default funding rate for testing
func NewFundingRateFixture() *FundingRateFixture {
	now := time.Now().Truncate(time.Hour)
	return &FundingRateFixture{
		fundingRate: market_data.FundingRate{
			Exchange:        "binance",
			Symbol:          "BTC/USDT",
			Timestamp:       now,
			FundingRate:     0.0001,
			NextFundingTime: now.Add(8 * time.Hour),
			MarkPrice:       50000.0,
			IndexPrice:      49995.0,
		},
	}
}

// WithExchange sets the exchange
func (f *FundingRateFixture) WithExchange(exchange string) *FundingRateFixture {
	f.fundingRate.Exchange = exchange
	return f
}

// WithSymbol sets the symbol
func (f *FundingRateFixture) WithSymbol(symbol string) *FundingRateFixture {
	f.fundingRate.Symbol = symbol
	return f
}

// WithTimestamp sets the timestamp
func (f *FundingRateFixture) WithTimestamp(t time.Time) *FundingRateFixture {
	f.fundingRate.Timestamp = t
	return f
}

// WithFundingRate sets the funding rate
func (f *FundingRateFixture) WithFundingRate(rate float64) *FundingRateFixture {
	f.fundingRate.FundingRate = rate
	return f
}

// WithMarkPrice sets the mark price
func (f *FundingRateFixture) WithMarkPrice(price float64) *FundingRateFixture {
	f.fundingRate.MarkPrice = price
	return f
}

// Build returns the constructed funding rate
func (f *FundingRateFixture) Build() market_data.FundingRate {
	return f.fundingRate
}

// ========================================
// Generic Helper Functions
// ========================================

// parseTimeframeDuration converts timeframe string to duration
func parseTimeframeDuration(timeframe string) time.Duration {
	switch timeframe {
	case "1m":
		return 1 * time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return 1 * time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return 1 * time.Hour
	}
}

// BuilderFunc is a generic function type for building test fixtures
type BuilderFunc[T any] func() T

// BuildMany is a generic helper to create multiple instances using a builder
func BuildMany[T any](builder BuilderFunc[T], count int) []T {
	items := make([]T, count)
	for i := 0; i < count; i++ {
		items[i] = builder()
	}
	return items
}

// BuildManyWith is a generic helper to create multiple instances with custom modifications
func BuildManyWith[T any](base T, count int, modifier func(T, int) T) []T {
	items := make([]T, count)
	for i := 0; i < count; i++ {
		items[i] = modifier(base, i)
	}
	return items
}

// OHLCVSequence creates a sequence of candles with incrementing time
func OHLCVSequence(base market_data.OHLCV, count int, timeframe string) []market_data.OHLCV {
	duration := parseTimeframeDuration(timeframe)
	return BuildManyWith(base, count, func(candle market_data.OHLCV, i int) market_data.OHLCV {
		candle.OpenTime = base.OpenTime.Add(time.Duration(i) * duration)
		candle.CloseTime = candle.OpenTime.Add(duration)
		return candle
	})
}

// TradeSequence creates a sequence of trades with incrementing time and IDs
func TradeSequence(base market_data.Trade, count int) []market_data.Trade {
	return BuildManyWith(base, count, func(trade market_data.Trade, i int) market_data.Trade {
		trade.Timestamp = base.Timestamp.Add(time.Duration(i) * time.Millisecond)
		trade.TradeID = fmt.Sprintf("%s_%d", base.TradeID, i)
		return trade
	})
}
