package clickhouse

/*
Integration tests for MarketDataRepository

These tests verify the ClickHouse repository implementation for market data storage.
All tests use fixtures from internal/testsupport to create clean, maintainable test data.

Test Coverage:
  - OHLCV: Insert, GetLatest, GetOHLCV with filters, taker buy volume calculation
  - Tickers: Insert (single & batch), GetLatest with multiple entries
  - Trades: Insert, GetRecent, buy/sell side verification
  - Funding Rates: Insert, GetLatest with multiple entries
  - Mark Prices: Insert, GetLatest, GetHistory with limits
  - Order Books: Insert, GetLatest with multiple snapshots
  - Liquidations: Insert, GetRecent, value calculation, limits
  - Open Interest: Insert, GetLatest with multiple entries

Running Tests:
  1. Unit tests (no DB required):
     go test -short ./internal/repository/clickhouse/...

  2. Integration tests (requires ClickHouse):
     make docker-up  # Start ClickHouse
     go test -v ./internal/repository/clickhouse/...

  3. Specific test:
     go test -v -run TestMarketDataRepository_MarkPrice ./internal/repository/clickhouse/...

Note: Tests use test_exchange to isolate data and automatically clean up after execution.
*/

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/market_data"
	"prometheus/internal/testsupport"
)

func TestMarketDataRepository_InsertAndGetOHLCV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	// Create repository
	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	// Register cleanup for test data
	helper.RegisterTableCleanup(t, "ohlcv", "exchange IN ('binance', 'bybit', 'okx', 'test_exchange')")

	t.Run("InsertOHLCV_Success", func(t *testing.T) {
		// Use fixtures for clean test data creation
		btcCandle := testsupport.NewOHLCVFixture().
			WithSymbol("BTC/USDT").
			WithPrices(50000, 51000, 49500, 50500).
			WithVolume(100.5, 5025000).
			WithTrades(1500).
			WithBuyPressure(55). // 55% buy pressure
			Build()

		ethCandle := testsupport.NewOHLCVFixture().
			WithSymbol("ETH/USDT").
			WithPrices(3000, 3100, 2950, 3050).
			WithVolume(500, 1500000).
			WithTrades(800).
			WithBuyPressure(56). // 56% buy pressure
			Build()

		candles := []market_data.OHLCV{btcCandle, ethCandle}

		// Test the repository insert method
		err := repo.InsertOHLCV(ctx, candles)
		require.NoError(t, err)

		// Verify data was inserted
		var result []struct {
			Count uint64 `ch:"count()"`
		}
		err = helper.Client().Query(ctx, &result, "SELECT count() FROM ohlcv WHERE exchange = 'binance' AND symbol IN ('BTC/USDT', 'ETH/USDT')")
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.GreaterOrEqual(t, result[0].Count, uint64(2))
	})

	t.Run("InsertOHLCV_EmptySlice", func(t *testing.T) {
		err := repo.InsertOHLCV(ctx, []market_data.OHLCV{})
		require.NoError(t, err)
	})

	t.Run("GetLatestOHLCV_Success", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)

		// Setup: Create test data using fixtures
		candles := testsupport.NewOHLCVFixture().
			WithExchange("bybit").
			WithSymbol("SOL/USDT").
			WithTimeframe("5m").
			WithOpenTime(baseTime.Add(-10*time.Minute)).
			WithPrices(100, 102, 99, 101).
			WithVolume(1000, 100500).
			WithTrades(250).
			WithBuyPressure(60). // 60% buy pressure
			BuildMany(2)         // Create 2 sequential candles

		// Insert using generic batch helper
		testsupport.CreateBatch(t, helper, testsupport.InsertOHLCV, candles)

		// Test: Get latest 1 candle
		result, err := repo.GetLatestOHLCV(ctx, "bybit", "SOL/USDT", "5m", 1)
		require.NoError(t, err)
		require.Len(t, result, 1)

		// Verify: Should return the most recent candle
		assert.Equal(t, "bybit", result[0].Exchange)
		assert.Equal(t, "SOL/USDT", result[0].Symbol)
		assert.Equal(t, "5m", result[0].Timeframe)
		assert.Equal(t, uint64(250), result[0].Trades)

		// Test: Get latest 2 candles
		result, err = repo.GetLatestOHLCV(ctx, "bybit", "SOL/USDT", "5m", 2)
		require.NoError(t, err)
		require.Len(t, result, 2)
	})

	t.Run("GetOHLCV_WithFilters", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Hour)

		// Insert test data across multiple hours
		candles := []market_data.OHLCV{
			{
				Exchange:   "okx",
				Symbol:     "BNB/USDT",
				Timeframe:  "1h",
				MarketType: "spot",
				OpenTime:   baseTime.Add(-3 * time.Hour),
				CloseTime:  baseTime.Add(-2 * time.Hour),
				Open:       300.0,
				High:       310.0,
				Low:        295.0,
				Close:      305.0,
				Volume:     500.0,
				IsClosed:   true,
				EventTime:  baseTime.Add(-3 * time.Hour),
			},
			{
				Exchange:   "okx",
				Symbol:     "BNB/USDT",
				Timeframe:  "1h",
				MarketType: "spot",
				OpenTime:   baseTime.Add(-2 * time.Hour),
				CloseTime:  baseTime.Add(-1 * time.Hour),
				Open:       305.0,
				High:       315.0,
				Low:        302.0,
				Close:      310.0,
				Volume:     600.0,
				IsClosed:   true,
				EventTime:  baseTime.Add(-2 * time.Hour),
			},
			{
				Exchange:   "okx",
				Symbol:     "BNB/USDT",
				Timeframe:  "1h",
				MarketType: "spot",
				OpenTime:   baseTime.Add(-1 * time.Hour),
				CloseTime:  baseTime,
				Open:       310.0,
				High:       320.0,
				Low:        308.0,
				Close:      315.0,
				Volume:     700.0,
				IsClosed:   true,
				EventTime:  baseTime.Add(-1 * time.Hour),
			},
		}

		err := repo.InsertOHLCV(ctx, candles)
		require.NoError(t, err)

		// Query with time range
		query := market_data.OHLCVQuery{
			Exchange:  "okx",
			Symbol:    "BNB/USDT",
			Timeframe: "1h",
			StartTime: baseTime.Add(-2*time.Hour - 1*time.Minute), // Include second candle (01:00)
			EndTime:   baseTime.Add(-90 * time.Minute),            // Exclude last candle (01:30 < 02:00)
			Limit:     10,
		}

		result, err := repo.GetOHLCV(ctx, query)
		require.NoError(t, err)
		require.Len(t, result, 1) // Should only return the second candle
		assert.Equal(t, 310.0, result[0].Close)
	})

	t.Run("TakerBuyVolume_Calculation", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)

		candle := market_data.OHLCV{
			Exchange:            "binance",
			Symbol:              "XRP/USDT",
			Timeframe:           "1m",
			MarketType:          "spot",
			OpenTime:            baseTime,
			CloseTime:           baseTime.Add(1 * time.Minute),
			Open:                0.5,
			High:                0.52,
			Low:                 0.49,
			Close:               0.51,
			Volume:              1000000.0,
			QuoteVolume:         500000.0,
			Trades:              5000,
			TakerBuyBaseVolume:  650000.0, // 65% buying pressure
			TakerBuyQuoteVolume: 325000.0,
			IsClosed:            true,
			EventTime:           baseTime,
		}

		err := repo.InsertOHLCV(ctx, []market_data.OHLCV{candle})
		require.NoError(t, err)

		result, err := repo.GetLatestOHLCV(ctx, "binance", "XRP/USDT", "1m", 1)
		require.NoError(t, err)
		require.Len(t, result, 1)

		// Verify taker buy volume fields
		assert.Equal(t, 650000.0, result[0].TakerBuyBaseVolume)
		assert.Equal(t, 325000.0, result[0].TakerBuyQuoteVolume)

		// Calculate buy pressure percentage
		buyPressure := (result[0].TakerBuyBaseVolume / result[0].Volume) * 100
		assert.InDelta(t, 65.0, buyPressure, 0.1) // 65% buy pressure
	})
}

func TestMarketDataRepository_Ticker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	// Register cleanup for test data
	helper.RegisterTableCleanup(t, "tickers", "exchange = 'test_exchange'")

	t.Run("InsertAndGetLatestTicker", func(t *testing.T) {
		// Use fixture for clean test data creation
		ticker := testsupport.NewTickerFixture().
			WithExchange("test_exchange").
			WithSymbol("BTC/USDT").
			WithPrice(50000.0).
			Build()

		err := repo.InsertTicker(ctx, []market_data.Ticker{ticker})
		require.NoError(t, err)

		result, err := repo.GetLatestTicker(ctx, "test_exchange", "BTC/USDT")
		require.NoError(t, err)
		assert.Equal(t, ticker.LastPrice, result.LastPrice)
		assert.Equal(t, ticker.Volume, result.Volume)
		assert.Equal(t, ticker.PriceChangePercent, result.PriceChangePercent)
	})

	t.Run("InsertTicker_EmptySlice", func(t *testing.T) {
		err := repo.InsertTicker(ctx, []market_data.Ticker{})
		require.NoError(t, err)
	})

	t.Run("InsertTicker_MultipleTickers", func(t *testing.T) {
		tickers := []market_data.Ticker{
			testsupport.NewTickerFixture().
				WithExchange("test_exchange").
				WithSymbol("ETH/USDT").
				WithPrice(3000.0).
				Build(),
			testsupport.NewTickerFixture().
				WithExchange("test_exchange").
				WithSymbol("SOL/USDT").
				WithPrice(100.0).
				Build(),
		}

		err := repo.InsertTicker(ctx, tickers)
		require.NoError(t, err)

		// Verify ETH ticker
		ethTicker, err := repo.GetLatestTicker(ctx, "test_exchange", "ETH/USDT")
		require.NoError(t, err)
		assert.Equal(t, 3000.0, ethTicker.LastPrice)

		// Verify SOL ticker
		solTicker, err := repo.GetLatestTicker(ctx, "test_exchange", "SOL/USDT")
		require.NoError(t, err)
		assert.Equal(t, 100.0, solTicker.LastPrice)
	})
}

func TestMarketDataRepository_Trades(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	// Register cleanup for test data
	helper.RegisterTableCleanup(t, "trades", "exchange = 'test_exchange'")

	t.Run("InsertAndGetRecentTrades", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Second)

		// Use fixtures for clean test data creation
		trades := []market_data.Trade{
			testsupport.NewTradeFixture().
				WithExchange("test_exchange").
				WithSymbol("ETH/USDT").
				WithTimestamp(baseTime.Add(-2 * time.Second)).
				WithTradeID(12345).
				WithPrice(3000.0).
				WithQuantity(1.5).
				AsBuy().
				Build(),
			testsupport.NewTradeFixture().
				WithExchange("test_exchange").
				WithSymbol("ETH/USDT").
				WithTimestamp(baseTime.Add(-1 * time.Second)).
				WithTradeID(12346).
				WithPrice(3001.0).
				WithQuantity(2.0).
				AsSell().
				Build(),
		}

		err := repo.InsertTrades(ctx, trades)
		require.NoError(t, err)

		result, err := repo.GetRecentTrades(ctx, "test_exchange", "ETH/USDT", 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result), 2)

		// Most recent trade should be first (ordered DESC by timestamp)
		assert.Equal(t, 3001.0, result[0].Price)
		assert.Equal(t, int64(12346), result[0].TradeID)
		assert.True(t, result[0].IsBuyerMaker) // Sell trade = buyer was maker
	})

	t.Run("InsertTrades_EmptySlice", func(t *testing.T) {
		err := repo.InsertTrades(ctx, []market_data.Trade{})
		require.NoError(t, err)
	})

	t.Run("InsertTrades_BuyVsSellSide", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Second)

		buyTrade := testsupport.NewTradeFixture().
			WithExchange("test_exchange").
			WithSymbol("BTC/USDT").
			WithTimestamp(baseTime).
			WithTradeID(99999).
			WithPrice(50000.0).
			AsBuy().
			Build()

		sellTrade := testsupport.NewTradeFixture().
			WithExchange("test_exchange").
			WithSymbol("BTC/USDT").
			WithTimestamp(baseTime.Add(1 * time.Second)).
			WithTradeID(100000).
			WithPrice(50001.0).
			AsSell().
			Build()

		err := repo.InsertTrades(ctx, []market_data.Trade{buyTrade, sellTrade})
		require.NoError(t, err)

		result, err := repo.GetRecentTrades(ctx, "test_exchange", "BTC/USDT", 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result), 2)

		// Verify buy/sell flag
		for _, trade := range result {
			if trade.TradeID == 99999 {
				assert.False(t, trade.IsBuyerMaker, "Buy trade should have IsBuyerMaker=false")
			}
			if trade.TradeID == 100000 {
				assert.True(t, trade.IsBuyerMaker, "Sell trade should have IsBuyerMaker=true")
			}
		}
	})
}

func TestMarketDataRepository_FundingRate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	// Register cleanup for test data
	helper.RegisterTableCleanup(t, "funding_rates", "exchange = 'test_exchange'")

	t.Run("InsertAndGetLatestFundingRate", func(t *testing.T) {
		// Use fixture for clean test data creation
		fundingRate := testsupport.NewFundingRateFixture().
			WithExchange("test_exchange").
			WithSymbol("BTC/USDT").
			WithFundingRate(0.0001).
			WithMarkPrice(50000.0).
			Build()

		err := repo.InsertFundingRate(ctx, &fundingRate)
		require.NoError(t, err)

		result, err := repo.GetLatestFundingRate(ctx, "test_exchange", "BTC/USDT")
		require.NoError(t, err)
		assert.Equal(t, fundingRate.FundingRate, result.FundingRate)
		assert.Equal(t, fundingRate.MarkPrice, result.MarkPrice)
		assert.Equal(t, fundingRate.IndexPrice, result.IndexPrice)
	})

	t.Run("GetLatestFundingRate_MultipleEntries", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Hour)

		// Insert older funding rate
		oldFundingRate := testsupport.NewFundingRateFixture().
			WithExchange("test_exchange").
			WithSymbol("ETH/USDT").
			WithTimestamp(baseTime.Add(-1 * time.Hour)).
			WithFundingRate(0.0002).
			WithMarkPrice(2990.0).
			Build()

		err := repo.InsertFundingRate(ctx, &oldFundingRate)
		require.NoError(t, err)

		// Insert newer funding rate
		newFundingRate := testsupport.NewFundingRateFixture().
			WithExchange("test_exchange").
			WithSymbol("ETH/USDT").
			WithTimestamp(baseTime).
			WithFundingRate(0.00025).
			WithMarkPrice(3000.0).
			Build()

		err = repo.InsertFundingRate(ctx, &newFundingRate)
		require.NoError(t, err)

		// Should return the latest one
		result, err := repo.GetLatestFundingRate(ctx, "test_exchange", "ETH/USDT")
		require.NoError(t, err)
		assert.Equal(t, 0.00025, result.FundingRate)
		assert.Equal(t, 3000.0, result.MarkPrice)
	})
}

func TestMarketDataRepository_MarkPrice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	// Register cleanup for test data
	helper.RegisterTableCleanup(t, "mark_price", "exchange = 'test_exchange'")

	t.Run("InsertAndGetLatestMarkPrice", func(t *testing.T) {
		// Use fixture for clean test data creation
		markPrice := testsupport.NewMarkPriceFixture().
			WithExchange("test_exchange").
			WithSymbol("BTC/USDT").
			WithMarketType("linear_perp").
			WithMarkPrice(50000.0).
			WithIndexPrice(49995.0).
			WithFundingRate(0.0001).
			Build()

		err := repo.InsertMarkPrice(ctx, []market_data.MarkPrice{markPrice})
		require.NoError(t, err)

		result, err := repo.GetLatestMarkPrice(ctx, "test_exchange", "BTC/USDT")
		require.NoError(t, err)
		assert.Equal(t, markPrice.MarkPrice, result.MarkPrice)
		assert.Equal(t, markPrice.IndexPrice, result.IndexPrice)
		assert.Equal(t, markPrice.FundingRate, result.FundingRate)
		assert.Equal(t, markPrice.MarketType, result.MarketType)
	})

	t.Run("InsertMarkPrice_EmptySlice", func(t *testing.T) {
		err := repo.InsertMarkPrice(ctx, []market_data.MarkPrice{})
		require.NoError(t, err)
	})

	t.Run("GetMarkPriceHistory", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Second)

		// Create sequence of mark prices using fixture
		markPrices := testsupport.NewMarkPriceFixture().
			WithExchange("test_exchange").
			WithSymbol("ETH/USDT").
			WithMarketType("linear_perp").
			WithTimestamp(baseTime.Add(-10 * time.Second)).
			BuildMany(5) // Create 5 sequential entries

		err := repo.InsertMarkPrice(ctx, markPrices)
		require.NoError(t, err)

		// Get history
		result, err := repo.GetMarkPriceHistory(ctx, "test_exchange", "ETH/USDT", 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result), 5)

		// Verify order (should be DESC by timestamp)
		for i := 0; i < len(result)-1; i++ {
			assert.True(t, result[i].Timestamp.After(result[i+1].Timestamp) || result[i].Timestamp.Equal(result[i+1].Timestamp),
				"Mark prices should be ordered DESC by timestamp")
		}
	})

	t.Run("GetMarkPriceHistory_WithLimit", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Second)

		markPrices := testsupport.NewMarkPriceFixture().
			WithExchange("test_exchange").
			WithSymbol("SOL/USDT").
			WithTimestamp(baseTime).
			BuildMany(10)

		err := repo.InsertMarkPrice(ctx, markPrices)
		require.NoError(t, err)

		// Request only 3 most recent
		result, err := repo.GetMarkPriceHistory(ctx, "test_exchange", "SOL/USDT", 3)
		require.NoError(t, err)
		require.Len(t, result, 3)
	})
}

func TestMarketDataRepository_OrderBook(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	// Register cleanup for test data
	helper.RegisterTableCleanup(t, "orderbook_snapshots", "exchange = 'test_exchange'")

	t.Run("InsertAndGetLatestOrderBook", func(t *testing.T) {
		// Use fixture for clean test data creation
		snapshot := testsupport.NewOrderBookSnapshotFixture().
			WithExchange("test_exchange").
			WithSymbol("BTC/USDT").
			WithMarketType("spot").
			WithBids(`[[49990.0, 1.5], [49980.0, 2.0]]`).
			WithAsks(`[[50010.0, 1.2], [50020.0, 2.5]]`).
			WithDepth(3.5, 3.7).
			Build()

		err := repo.InsertOrderBook(ctx, []market_data.OrderBookSnapshot{snapshot})
		require.NoError(t, err)

		result, err := repo.GetLatestOrderBook(ctx, "test_exchange", "BTC/USDT")
		require.NoError(t, err)
		assert.Equal(t, snapshot.Exchange, result.Exchange)
		assert.Equal(t, snapshot.Symbol, result.Symbol)
		assert.Equal(t, snapshot.BidDepth, result.BidDepth)
		assert.Equal(t, snapshot.AskDepth, result.AskDepth)
		assert.Contains(t, result.Bids, "49990.0")
		assert.Contains(t, result.Asks, "50010.0")
	})

	t.Run("InsertOrderBook_EmptySlice", func(t *testing.T) {
		err := repo.InsertOrderBook(ctx, []market_data.OrderBookSnapshot{})
		require.NoError(t, err)
	})

	t.Run("GetLatestOrderBook_MultipleSnapshots", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Second)

		// Insert older snapshot
		oldSnapshot := testsupport.NewOrderBookSnapshotFixture().
			WithExchange("test_exchange").
			WithSymbol("ETH/USDT").
			WithTimestamp(baseTime.Add(-5 * time.Second)).
			WithBids(`[[2990.0, 10.0]]`).
			WithDepth(10.0, 8.0).
			Build()

		err := repo.InsertOrderBook(ctx, []market_data.OrderBookSnapshot{oldSnapshot})
		require.NoError(t, err)

		// Insert newer snapshot
		newSnapshot := testsupport.NewOrderBookSnapshotFixture().
			WithExchange("test_exchange").
			WithSymbol("ETH/USDT").
			WithTimestamp(baseTime).
			WithBids(`[[3000.0, 15.0]]`).
			WithDepth(15.0, 12.0).
			Build()

		err = repo.InsertOrderBook(ctx, []market_data.OrderBookSnapshot{newSnapshot})
		require.NoError(t, err)

		// Should return the latest snapshot
		result, err := repo.GetLatestOrderBook(ctx, "test_exchange", "ETH/USDT")
		require.NoError(t, err)
		assert.Equal(t, 15.0, result.BidDepth)
		assert.Contains(t, result.Bids, "3000.0")
	})
}

func TestMarketDataRepository_Liquidations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	// Register cleanup for test data
	helper.RegisterTableCleanup(t, "liquidations", "exchange = 'test_exchange'")

	t.Run("InsertAndGetRecentLiquidations", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Second)

		// Use fixtures for clean test data creation
		liquidations := []market_data.Liquidation{
			testsupport.NewLiquidationFixture().
				WithExchange("test_exchange").
				WithSymbol("BTC/USDT").
				WithTimestamp(baseTime.Add(-2 * time.Second)).
				WithSide("LONG").
				WithPrice(49000.0).
				WithQuantity(2.5).
				Build(),
			testsupport.NewLiquidationFixture().
				WithExchange("test_exchange").
				WithSymbol("BTC/USDT").
				WithTimestamp(baseTime.Add(-1 * time.Second)).
				WithSide("SHORT").
				WithPrice(51000.0).
				WithQuantity(1.5).
				Build(),
		}

		err := repo.InsertLiquidations(ctx, liquidations)
		require.NoError(t, err)

		result, err := repo.GetRecentLiquidations(ctx, "test_exchange", "BTC/USDT", 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result), 2)

		// Most recent liquidation should be first
		assert.Equal(t, "SHORT", result[0].Side)
		assert.Equal(t, 51000.0, result[0].Price)
		assert.Equal(t, 1.5, result[0].Quantity)
	})

	t.Run("InsertLiquidations_EmptySlice", func(t *testing.T) {
		err := repo.InsertLiquidations(ctx, []market_data.Liquidation{})
		require.NoError(t, err)
	})

	t.Run("Liquidation_ValueCalculation", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Second)

		liq := testsupport.NewLiquidationFixture().
			WithExchange("test_exchange").
			WithSymbol("ETH/USDT").
			WithTimestamp(baseTime).
			AsLongLiquidation().
			WithPrice(3000.0).
			WithQuantity(10.0). // Value should be 30000
			Build()

		err := repo.InsertLiquidations(ctx, []market_data.Liquidation{liq})
		require.NoError(t, err)

		result, err := repo.GetRecentLiquidations(ctx, "test_exchange", "ETH/USDT", 1)
		require.NoError(t, err)
		require.Len(t, result, 1)

		// Verify value calculation
		expectedValue := 3000.0 * 10.0
		assert.Equal(t, expectedValue, result[0].Value)
		assert.Equal(t, "LONG", result[0].Side)
	})

	t.Run("GetRecentLiquidations_WithLimit", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Second)

		// Create multiple liquidations
		liquidations := testsupport.NewLiquidationFixture().
			WithExchange("test_exchange").
			WithSymbol("SOL/USDT").
			WithTimestamp(baseTime).
			BuildMany(10)

		err := repo.InsertLiquidations(ctx, liquidations)
		require.NoError(t, err)

		// Request only 3 most recent
		result, err := repo.GetRecentLiquidations(ctx, "test_exchange", "SOL/USDT", 3)
		require.NoError(t, err)
		require.Len(t, result, 3)
	})
}

func TestMarketDataRepository_OpenInterest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := testsupport.LoadDatabaseConfigsFromEnv(t)
	helper := testsupport.NewClickHouseTestHelper(t, cfg.ClickHouse)

	repo := NewMarketDataRepository(helper.Client().Conn())
	ctx := context.Background()

	// Register cleanup for test data
	helper.RegisterTableCleanup(t, "open_interest", "exchange = 'test_exchange'")

	t.Run("InsertAndGetLatestOpenInterest", func(t *testing.T) {
		// Use fixture for clean test data creation
		oi := testsupport.NewOpenInterestFixture().
			WithExchange("test_exchange").
			WithSymbol("BTC/USDT").
			WithAmount(1500000.0).
			Build()

		err := repo.InsertOpenInterest(ctx, &oi)
		require.NoError(t, err)

		result, err := repo.GetLatestOpenInterest(ctx, "test_exchange", "BTC/USDT")
		require.NoError(t, err)
		assert.Equal(t, oi.Amount, result.Amount)
		assert.Equal(t, oi.Exchange, result.Exchange)
		assert.Equal(t, oi.Symbol, result.Symbol)
	})

	t.Run("GetLatestOpenInterest_MultipleEntries", func(t *testing.T) {
		baseTime := time.Now().Truncate(time.Minute)

		// Insert older open interest
		oldOI := testsupport.NewOpenInterestFixture().
			WithExchange("test_exchange").
			WithSymbol("ETH/USDT").
			WithTimestamp(baseTime.Add(-5 * time.Minute)).
			WithAmount(500000.0).
			Build()

		err := repo.InsertOpenInterest(ctx, &oldOI)
		require.NoError(t, err)

		// Insert newer open interest
		newOI := testsupport.NewOpenInterestFixture().
			WithExchange("test_exchange").
			WithSymbol("ETH/USDT").
			WithTimestamp(baseTime).
			WithAmount(550000.0).
			Build()

		err = repo.InsertOpenInterest(ctx, &newOI)
		require.NoError(t, err)

		// Should return the latest one
		result, err := repo.GetLatestOpenInterest(ctx, "test_exchange", "ETH/USDT")
		require.NoError(t, err)
		assert.Equal(t, 550000.0, result.Amount)
	})
}
