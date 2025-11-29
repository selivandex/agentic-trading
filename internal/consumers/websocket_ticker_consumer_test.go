package consumers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/market_data"
)

func TestWebSocketTickerConsumer_DeduplicateBatch(t *testing.T) {
	consumer := &WebSocketTickerConsumer{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		input          []*market_data.Ticker
		expectedCount  int
		expectedSymbol string
		expectedPrice  float64 // Expected last price (from latest event)
		description    string
	}{
		{
			name: "no_duplicates",
			input: []*market_data.Ticker{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					LastPrice: 50000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "ETHUSDT",
					Timestamp: baseTime,
					LastPrice: 3000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
			},
			expectedCount: 2,
			description:   "Different symbols - no deduplication",
		},
		{
			name: "same_symbol_different_timestamps",
			input: []*market_data.Ticker{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					LastPrice: 50000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime.Add(1 * time.Second),
					LastPrice: 50100.0,
					EventTime: baseTime.Add(11 * time.Second),
				},
			},
			expectedCount: 2,
			description:   "Same symbol, different timestamps - no deduplication",
		},
		{
			name: "duplicate_updates_keep_latest",
			input: []*market_data.Ticker{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					LastPrice: 50000.0,
					EventTime: baseTime.Add(5 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					LastPrice: 50050.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					LastPrice: 50100.0,
					EventTime: baseTime.Add(15 * time.Second),
				},
			},
			expectedCount:  1,
			expectedSymbol: "BTCUSDT",
			expectedPrice:  50100.0, // Should keep the latest (15s)
			description:    "Multiple updates of same ticker - keep only latest",
		},
		{
			name: "mixed_duplicates_and_unique",
			input: []*market_data.Ticker{
				// BTCUSDT - 3 updates at same timestamp
				{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, LastPrice: 50000.0, EventTime: baseTime.Add(5 * time.Second)},
				{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, LastPrice: 50100.0, EventTime: baseTime.Add(10 * time.Second)},
				{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, LastPrice: 50200.0, EventTime: baseTime.Add(15 * time.Second)},

				// ETHUSDT - 2 updates at same timestamp
				{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, LastPrice: 3000.0, EventTime: baseTime.Add(5 * time.Second)},
				{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, LastPrice: 3010.0, EventTime: baseTime.Add(12 * time.Second)},

				// BTCUSDT at different timestamp - unique
				{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime.Add(1 * time.Second), LastPrice: 50150.0, EventTime: baseTime.Add(10 * time.Second)},
			},
			expectedCount: 3, // Should have 3 unique tickers after dedup
			description:   "Mixed scenario - should reduce from 6 to 3",
		},
		{
			name: "out_of_order_events_keep_latest_by_event_time",
			input: []*market_data.Ticker{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					LastPrice: 50200.0,
					EventTime: baseTime.Add(30 * time.Second), // Latest
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					LastPrice: 50000.0,
					EventTime: baseTime.Add(5 * time.Second), // Oldest
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					Timestamp: baseTime,
					LastPrice: 50100.0,
					EventTime: baseTime.Add(15 * time.Second), // Middle
				},
			},
			expectedCount:  1,
			expectedSymbol: "BTCUSDT",
			expectedPrice:  50200.0, // Should keep event with event_time = 30s
			description:    "Out-of-order events - keep latest by event_time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := consumer.deduplicateBatch(tt.input)

			// Check count
			assert.Equal(t, tt.expectedCount, len(result), tt.description)

			// If specific symbol/price expected, verify it
			if tt.expectedSymbol != "" {
				found := false
				for _, ticker := range result {
					if ticker.Symbol == tt.expectedSymbol {
						found = true
						assert.Equal(t, tt.expectedPrice, ticker.LastPrice,
							"Should keep the latest update (by event_time)")
					}
				}
				assert.True(t, found, "Expected symbol should be in result")
			}
		})
	}
}

func TestWebSocketTickerConsumer_DeduplicateBatch_NoDataLoss(t *testing.T) {
	consumer := &WebSocketTickerConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Simulate real scenario: 24hr ticker updates (usually every 1-3 seconds)
	input := []*market_data.Ticker{
		// BTCUSDT - 5 updates at same timestamp
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, LastPrice: 50000.0, Volume: 1000.0, EventTime: baseTime.Add(1 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, LastPrice: 50010.0, Volume: 1010.0, EventTime: baseTime.Add(2 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, LastPrice: 50020.0, Volume: 1020.0, EventTime: baseTime.Add(3 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, LastPrice: 50030.0, Volume: 1030.0, EventTime: baseTime.Add(4 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", Timestamp: baseTime, LastPrice: 50040.0, Volume: 1040.0, EventTime: baseTime.Add(5 * time.Second)},

		// ETHUSDT - 3 updates at same timestamp
		{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, LastPrice: 3000.0, EventTime: baseTime.Add(1 * time.Second)},
		{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, LastPrice: 3005.0, EventTime: baseTime.Add(3 * time.Second)},
		{Exchange: "binance", Symbol: "ETHUSDT", Timestamp: baseTime, LastPrice: 3010.0, EventTime: baseTime.Add(5 * time.Second)},

		// SOLUSDT - 1 update (no duplicates)
		{Exchange: "binance", Symbol: "SOLUSDT", Timestamp: baseTime, LastPrice: 100.0, EventTime: baseTime.Add(2 * time.Second)},
	}

	result := consumer.deduplicateBatch(input)

	// Should reduce from 9 events to 3 unique tickers
	require.Equal(t, 3, len(result), "Should have 3 unique tickers after dedup")

	// Verify each unique ticker has the latest value
	priceMap := make(map[string]float64)
	for _, ticker := range result {
		priceMap[ticker.Symbol] = ticker.LastPrice
	}

	assert.Equal(t, 50040.0, priceMap["BTCUSDT"], "BTCUSDT should have latest price (50040)")
	assert.Equal(t, 3010.0, priceMap["ETHUSDT"], "ETHUSDT should have latest price (3010)")
	assert.Equal(t, 100.0, priceMap["SOLUSDT"], "SOLUSDT should have price (100)")
}

func TestWebSocketTickerConsumer_DeduplicateBatch_EmptyBatch(t *testing.T) {
	consumer := &WebSocketTickerConsumer{}

	result := consumer.deduplicateBatch([]*market_data.Ticker{})
	assert.Empty(t, result, "Empty batch should return empty result")
}

func TestWebSocketTickerConsumer_DeduplicateBatch_PreservesAllFields(t *testing.T) {
	consumer := &WebSocketTickerConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.Ticker{
		{
			Exchange:           "binance",
			Symbol:             "BTCUSDT",
			MarketType:         "futures",
			Timestamp:          baseTime,
			LastPrice:          50000.0,
			OpenPrice:          49500.0,
			HighPrice:          50200.0,
			LowPrice:           49300.0,
			Volume:             1000000.0,
			QuoteVolume:        50000000000.0,
			PriceChange:        500.0,
			PriceChangePercent: 1.01,
			WeightedAvgPrice:   49950.0,
			TradeCount:         123456,
			EventTime:          baseTime.Add(30 * time.Second),
		},
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timestamp: baseTime,
			LastPrice: 50075.0,                        // Updated price
			EventTime: baseTime.Add(45 * time.Second), // Later event
		},
	}

	result := consumer.deduplicateBatch(input)

	require.Len(t, result, 1, "Should deduplicate to 1 ticker")

	ticker := result[0]
	assert.Equal(t, "binance", ticker.Exchange)
	assert.Equal(t, "BTCUSDT", ticker.Symbol)
	assert.Equal(t, 50075.0, ticker.LastPrice, "Should have latest price")
	assert.Equal(t, baseTime.Add(45*time.Second), ticker.EventTime, "Should have latest event time")
}

func TestWebSocketTickerConsumer_DeduplicateBatch_DifferentExchanges(t *testing.T) {
	consumer := &WebSocketTickerConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.Ticker{
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timestamp: baseTime,
			LastPrice: 50000.0,
			EventTime: baseTime.Add(10 * time.Second),
		},
		{
			Exchange:  "bybit",
			Symbol:    "BTCUSDT",
			Timestamp: baseTime,
			LastPrice: 50010.0,
			EventTime: baseTime.Add(10 * time.Second),
		},
	}

	result := consumer.deduplicateBatch(input)

	assert.Equal(t, 2, len(result), "Different exchanges should not deduplicate")
}

func TestWebSocketTickerConsumer_DeduplicateBatch_VolumeUpdates(t *testing.T) {
	consumer := &WebSocketTickerConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.Ticker{
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timestamp: baseTime,
			LastPrice: 50000.0,
			Volume:    1000.0,
			EventTime: baseTime.Add(10 * time.Second),
		},
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Timestamp: baseTime,
			LastPrice: 50050.0,
			Volume:    1500.0, // Volume increased
			EventTime: baseTime.Add(30 * time.Second),
		},
	}

	result := consumer.deduplicateBatch(input)

	require.Len(t, result, 1, "Should keep only 1 ticker")
	assert.Equal(t, 1500.0, result[0].Volume, "Should have latest volume")
	assert.Equal(t, 50050.0, result[0].LastPrice, "Should have latest price")
}

func TestWebSocketTickerConsumer_DeduplicateBatch_Performance(t *testing.T) {
	consumer := &WebSocketTickerConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Create large batch with many duplicates (realistic scenario)
	// 20 symbols × 30 updates = 600 events → should dedup to 20
	symbols := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "ADAUSDT",
		"DOTUSDT", "MATICUSDT", "LINKUSDT", "AVAXUSDT", "ATOMUSDT",
		"XRPUSDT", "DOGEUSDT", "SHIBUSDT", "LTCUSDT", "BCHUSDT",
		"XLMUSDT", "ALGOUSDT", "VETUSDT", "FILUSDT", "MANAUSDT",
	}
	updates := 30

	input := make([]*market_data.Ticker, 0, len(symbols)*updates)

	for _, symbol := range symbols {
		for i := 0; i < updates; i++ {
			input = append(input, &market_data.Ticker{
				Exchange:  "binance",
				Symbol:    symbol,
				Timestamp: baseTime,
				LastPrice: 50000.0 + float64(i*10),
				Volume:    1000.0 + float64(i*100),
				EventTime: baseTime.Add(time.Duration(i+1) * time.Second),
			})
		}
	}

	require.Len(t, input, 600, "Should have 600 input events")

	start := time.Now()
	result := consumer.deduplicateBatch(input)
	duration := time.Since(start)

	expectedUnique := len(symbols) // 20
	assert.Equal(t, expectedUnique, len(result), "Should deduplicate 600 → 20 events")
	assert.Less(t, duration.Milliseconds(), int64(15), "Deduplication should be fast (<15ms)")

	t.Logf("✅ Deduplicated %d → %d events in %v", len(input), len(result), duration)
}

