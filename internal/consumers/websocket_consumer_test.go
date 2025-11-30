package consumers

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/market_data"
)

// TestDeduplicateBatch_Klines tests kline deduplication logic
func TestDeduplicateBatch_Klines(t *testing.T) {
	c := &WebSocketConsumer{}

	now := time.Now()
	earlier := now.Add(-1 * time.Second)
	later := now.Add(1 * time.Second)

	tests := []struct {
		name          string
		input         []interface{}
		expectedCount int
		description   string
	}{
		{
			name: "no duplicates",
			input: []interface{}{
				market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  now,
					EventTime: now,
				},
				market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    "ETHUSDT",
					Timeframe: "1m",
					OpenTime:  now,
					EventTime: now,
				},
			},
			expectedCount: 2,
			description:   "Different symbols should not deduplicate",
		},
		{
			name: "same kline different event_time - keep latest",
			input: []interface{}{
				market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  now,
					Close:     100.0,
					EventTime: earlier,
				},
				market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  now,
					Close:     101.0, // Updated price
					EventTime: later,
				},
			},
			expectedCount: 1,
			description:   "Should keep only latest kline by event_time",
		},
		{
			name: "different intervals - no deduplication",
			input: []interface{}{
				market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  now,
					EventTime: now,
				},
				market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "5m",
					OpenTime:  now,
					EventTime: now,
				},
			},
			expectedCount: 2,
			description:   "Different timeframes should not deduplicate",
		},
		{
			name: "multiple duplicates - keep latest",
			input: []interface{}{
				market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  now,
					Close:     100.0,
					EventTime: earlier,
				},
				market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  now,
					Close:     101.0,
					EventTime: now,
				},
				market_data.OHLCV{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timeframe: "1m",
					OpenTime:  now,
					Close:     102.0, // Latest
					EventTime: later,
				},
			},
			expectedCount: 1,
			description:   "With 3 duplicates, should keep only the latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.deduplicateBatch(tt.input)
			assert.Equal(t, tt.expectedCount, len(result), tt.description)

			// If we expect 1 result and had duplicates, verify it's the latest
			if tt.expectedCount == 1 && len(tt.input) > 1 {
				resultKline, ok := result[0].(market_data.OHLCV)
				require.True(t, ok, "Result should be OHLCV type")

				// Find latest in input
				var latestInput market_data.OHLCV
				latestTime := time.Time{}
				for _, item := range tt.input {
					kline := item.(market_data.OHLCV)
					if kline.EventTime.After(latestTime) {
						latestTime = kline.EventTime
						latestInput = kline
					}
				}

				assert.Equal(t, latestInput.Close, resultKline.Close,
					"Should keep the kline with latest event_time")
			}
		})
	}
}

// TestDeduplicateBatch_Tickers tests ticker deduplication
func TestDeduplicateBatch_Tickers(t *testing.T) {
	c := &WebSocketConsumer{}

	now := time.Now()
	earlier := now.Add(-500 * time.Millisecond)
	later := now.Add(500 * time.Millisecond)

	tests := []struct {
		name          string
		input         []interface{}
		expectedCount int
	}{
		{
			name: "same ticker updates - keep latest",
			input: []interface{}{
				market_data.Ticker{
					Exchange:   "binance",
					Symbol:     "BTCUSDT",
					MarketType: "futures",
					Timestamp:  now,
					LastPrice:  50000.0,
					EventTime:  earlier,
				},
				market_data.Ticker{
					Exchange:   "binance",
					Symbol:     "BTCUSDT",
					MarketType: "futures",
					Timestamp:  now,
					LastPrice:  50100.0,
					EventTime:  later,
				},
			},
			expectedCount: 1,
		},
		{
			name: "different market types - no dedup",
			input: []interface{}{
				market_data.Ticker{
					Exchange:   "binance",
					Symbol:     "BTCUSDT",
					MarketType: "spot",
					Timestamp:  now,
					EventTime:  now,
				},
				market_data.Ticker{
					Exchange:   "binance",
					Symbol:     "BTCUSDT",
					MarketType: "futures",
					Timestamp:  now,
					EventTime:  now,
				},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.deduplicateBatch(tt.input)
			assert.Equal(t, tt.expectedCount, len(result))
		})
	}
}

// TestDeduplicateBatch_MixedTypes tests mixed batch deduplication
func TestDeduplicateBatch_MixedTypes(t *testing.T) {
	c := &WebSocketConsumer{}

	now := time.Now()
	earlier := now.Add(-1 * time.Second)
	later := now.Add(1 * time.Second)

	input := []interface{}{
		// 2 duplicate klines (keep latest)
		market_data.OHLCV{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  now,
			EventTime: earlier,
		},
		market_data.OHLCV{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  now,
			EventTime: later,
		},
		// 2 duplicate tickers (keep latest)
		market_data.Ticker{
			Exchange:   "binance",
			Symbol:     "ETHUSDT",
			MarketType: "futures",
			Timestamp:  now,
			EventTime:  earlier,
		},
		market_data.Ticker{
			Exchange:   "binance",
			Symbol:     "ETHUSDT",
			MarketType: "futures",
			Timestamp:  now,
			EventTime:  later,
		},
		// 1 unique mark price
		market_data.MarkPrice{
			Exchange:  "binance",
			Symbol:    "SOLUSDT",
			Timestamp: now,
			EventTime: now,
		},
		// 1 unique trade
		market_data.Trade{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			TradeID:   12345,
			EventTime: now,
		},
	}

	result := c.deduplicateBatch(input)

	// Should have 4 items: 1 kline, 1 ticker, 1 mark price, 1 trade
	assert.Equal(t, 4, len(result), "Mixed batch should deduplicate per type")

	// Count types
	var klineCount, tickerCount, markPriceCount, tradeCount int
	for _, item := range result {
		switch item.(type) {
		case market_data.OHLCV:
			klineCount++
		case market_data.Ticker:
			tickerCount++
		case market_data.MarkPrice:
			markPriceCount++
		case market_data.Trade:
			tradeCount++
		}
	}

	assert.Equal(t, 1, klineCount, "Should have 1 deduplicated kline")
	assert.Equal(t, 1, tickerCount, "Should have 1 deduplicated ticker")
	assert.Equal(t, 1, markPriceCount, "Should have 1 mark price")
	assert.Equal(t, 1, tradeCount, "Should have 1 trade")
}

// TestDeduplicateBatch_Trades tests trade deduplication by TradeID
func TestDeduplicateBatch_Trades(t *testing.T) {
	c := &WebSocketConsumer{}

	now := time.Now()

	input := []interface{}{
		market_data.Trade{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			TradeID:   12345,
			Price:     50000.0,
			EventTime: now.Add(-1 * time.Second),
		},
		market_data.Trade{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			TradeID:   12345, // Same trade ID
			Price:     50001.0,
			EventTime: now, // Later event
		},
		market_data.Trade{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			TradeID:   12346, // Different trade ID
			Price:     50002.0,
			EventTime: now,
		},
	}

	result := c.deduplicateBatch(input)

	// Should have 2 trades (12345 deduplicated, 12346 unique)
	assert.Equal(t, 2, len(result), "Should deduplicate by TradeID")

	// Verify the latest version of trade 12345 was kept
	for _, item := range result {
		trade := item.(market_data.Trade)
		if trade.TradeID == 12345 {
			assert.Equal(t, 50001.0, trade.Price, "Should keep latest trade by event_time")
		}
	}
}

// TestDeduplicationKeys tests key generation functions
func TestDeduplicationKeys(t *testing.T) {
	now := time.Now()

	t.Run("kline key", func(t *testing.T) {
		kline := market_data.OHLCV{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  now,
		}
		key := getKlineKey(kline)
		expected := "binance_BTCUSDT_1m_" + strconv.FormatInt(now.Unix(), 10)
		assert.Equal(t, expected, key)
	})

	t.Run("ticker key", func(t *testing.T) {
		ticker := market_data.Ticker{
			Exchange:   "binance",
			Symbol:     "ETHUSDT",
			MarketType: "futures",
			Timestamp:  now,
		}
		key := getTickerKey(ticker)
		expected := "binance_ETHUSDT_futures_" + strconv.FormatInt(now.Unix(), 10)
		assert.Equal(t, expected, key)
	})

	t.Run("trade key", func(t *testing.T) {
		trade := market_data.Trade{
			Exchange: "binance",
			Symbol:   "BTCUSDT",
			TradeID:  12345,
		}
		key := getTradeKey(trade)
		expected := "binance_BTCUSDT_12345"
		assert.Equal(t, expected, key)
	})

	t.Run("mark price key", func(t *testing.T) {
		markPrice := market_data.MarkPrice{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timestamp: now,
		}
		key := getMarkPriceKey(markPrice)
		expected := "binance_BTCUSDT_" + strconv.FormatInt(now.Unix(), 10)
		assert.Equal(t, expected, key)
	})
}

// TestDeduplicateBatch_EmptyBatch tests edge case
func TestDeduplicateBatch_EmptyBatch(t *testing.T) {
	c := &WebSocketConsumer{}

	result := c.deduplicateBatch([]interface{}{})
	assert.Equal(t, 0, len(result), "Empty batch should return empty")
}

// TestDeduplicateBatch_Performance tests deduplication doesn't slow down significantly
func TestDeduplicateBatch_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	c := &WebSocketConsumer{}

	// Create batch with 1000 items, 50% duplicates
	now := time.Now()
	batch := make([]interface{}, 0, 1000)

	for i := 0; i < 500; i++ {
		openTime := now.Add(time.Duration(i) * time.Minute)
		// Add each kline twice (duplicate)
		for j := 0; j < 2; j++ {
			batch = append(batch, market_data.OHLCV{
				Exchange:  "binance",
				Symbol:    "BTCUSDT",
				Timeframe: "1m",
				OpenTime:  openTime,
				EventTime: now.Add(time.Duration(j) * time.Millisecond),
			})
		}
	}

	start := time.Now()
	result := c.deduplicateBatch(batch)
	duration := time.Since(start)

	assert.Equal(t, 500, len(result), "Should deduplicate 1000 -> 500")
	assert.Less(t, duration, 10*time.Millisecond,
		"Deduplication of 1000 items should be fast (<10ms)")

	t.Logf("Deduplicated 1000 items (50%% duplicates) in %v", duration)
}

// TestDeduplicateBatch_OrderMatters tests that later events win
func TestDeduplicateBatch_OrderMatters(t *testing.T) {
	c := &WebSocketConsumer{}

	now := time.Now()

	// Add in reverse chronological order
	input := []interface{}{
		market_data.OHLCV{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  now,
			Close:     103.0,
			EventTime: now.Add(2 * time.Second), // Latest
		},
		market_data.OHLCV{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  now,
			Close:     102.0,
			EventTime: now.Add(1 * time.Second),
		},
		market_data.OHLCV{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timeframe: "1m",
			OpenTime:  now,
			Close:     101.0,
			EventTime: now, // Oldest
		},
	}

	result := c.deduplicateBatch(input)

	require.Equal(t, 1, len(result))
	kline := result[0].(market_data.OHLCV)
	assert.Equal(t, 103.0, kline.Close, "Should keep the one with latest event_time")
}

// BenchmarkDeduplicateBatch benchmarks deduplication performance
func BenchmarkDeduplicateBatch(b *testing.B) {
	c := &WebSocketConsumer{}

	now := time.Now()

	// Create realistic batch: 100 items, 20% duplicates
	batch := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		symbolIndex := i % 80 // 80 unique, 20 duplicates
		batch[i] = market_data.OHLCV{
			Exchange:  "binance",
			Symbol:    "SYMBOL" + strconv.Itoa(symbolIndex),
			Timeframe: "1m",
			OpenTime:  now.Add(time.Duration(symbolIndex) * time.Minute),
			EventTime: now.Add(time.Duration(i) * time.Millisecond),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.deduplicateBatch(batch)
	}
}
