package consumers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"prometheus/internal/domain/market_data"
)

func TestWebSocketTradeConsumer_DeduplicateBatch(t *testing.T) {
	consumer := &WebSocketTradeConsumer{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		input          []*market_data.Trade
		expectedCount  int
		expectedSymbol string
		expectedPrice  float64 // Expected price (from latest event)
		description    string
	}{
		{
			name: "no_duplicates",
			input: []*market_data.Trade{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					TradeID:   1001,
					Price:     50000.0,
					Quantity:  1.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					TradeID:   1002,
					Price:     50010.0,
					Quantity:  0.5,
					EventTime: baseTime.Add(11 * time.Second),
				},
			},
			expectedCount: 2,
			description:   "Different trade IDs - no deduplication",
		},
		{
			name: "different_symbols_same_trade_id",
			input: []*market_data.Trade{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					TradeID:   1001,
					Price:     50000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "ETHUSDT",
					TradeID:   1001, // Same ID but different symbol
					Price:     3000.0,
					EventTime: baseTime.Add(10 * time.Second),
				},
			},
			expectedCount: 2,
			description:   "Same trade ID, different symbols - no deduplication",
		},
		{
			name: "duplicate_trade_id_keep_latest",
			input: []*market_data.Trade{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					TradeID:   1001,
					Price:     50000.0,
					Quantity:  1.0,
					EventTime: baseTime.Add(5 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					TradeID:   1001, // Duplicate trade ID (should not happen in practice)
					Price:     50050.0,
					Quantity:  1.5,
					EventTime: baseTime.Add(10 * time.Second),
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					TradeID:   1001,
					Price:     50100.0,
					Quantity:  2.0,
					EventTime: baseTime.Add(15 * time.Second),
				},
			},
			expectedCount:  1,
			expectedSymbol: "BTCUSDT",
			expectedPrice:  50100.0, // Should keep the latest (15s)
			description:    "Duplicate trade IDs - keep only latest by event_time",
		},
		{
			name: "mixed_symbols_and_trades",
			input: []*market_data.Trade{
				// BTCUSDT trades
				{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1001, Price: 50000.0, EventTime: baseTime.Add(1 * time.Second)},
				{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1002, Price: 50010.0, EventTime: baseTime.Add(2 * time.Second)},
				{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1003, Price: 50020.0, EventTime: baseTime.Add(3 * time.Second)},

				// ETHUSDT trades
				{Exchange: "binance", Symbol: "ETHUSDT", TradeID: 2001, Price: 3000.0, EventTime: baseTime.Add(1 * time.Second)},
				{Exchange: "binance", Symbol: "ETHUSDT", TradeID: 2002, Price: 3005.0, EventTime: baseTime.Add(2 * time.Second)},

				// Duplicate BTCUSDT trade (edge case)
				{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1002, Price: 50015.0, EventTime: baseTime.Add(4 * time.Second)},
			},
			expectedCount: 5, // Should have 5 unique trades (1001, 1002 latest, 1003, 2001, 2002)
			description:   "Mixed scenario - should reduce from 6 to 5",
		},
		{
			name: "out_of_order_events_keep_latest_by_event_time",
			input: []*market_data.Trade{
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					TradeID:   1001,
					Price:     50200.0,
					EventTime: baseTime.Add(30 * time.Second), // Latest
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					TradeID:   1001,
					Price:     50000.0,
					EventTime: baseTime.Add(5 * time.Second), // Oldest
				},
				{
					Exchange:  "binance",
					Symbol:    "BTCUSDT",
					TradeID:   1001,
					Price:     50100.0,
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
			if tt.expectedSymbol != "" && tt.expectedCount == 1 {
				found := false
				for _, trade := range result {
					if trade.Symbol == tt.expectedSymbol {
						found = true
						assert.Equal(t, tt.expectedPrice, trade.Price,
							"Should keep the latest update (by event_time)")
					}
				}
				assert.True(t, found, "Expected symbol should be in result")
			}
		})
	}
}

func TestWebSocketTradeConsumer_DeduplicateBatch_NoDataLoss(t *testing.T) {
	consumer := &WebSocketTradeConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Simulate real scenario: high-frequency trades
	input := []*market_data.Trade{
		// BTCUSDT trades - sequential trade IDs (normal case)
		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1001, Price: 50000.0, Quantity: 1.0, EventTime: baseTime.Add(1 * time.Millisecond)},
		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1002, Price: 50001.0, Quantity: 0.5, EventTime: baseTime.Add(2 * time.Millisecond)},
		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1003, Price: 50002.0, Quantity: 2.0, EventTime: baseTime.Add(3 * time.Millisecond)},
		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1004, Price: 50003.0, Quantity: 0.3, EventTime: baseTime.Add(4 * time.Millisecond)},
		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1005, Price: 50004.0, Quantity: 1.5, EventTime: baseTime.Add(5 * time.Millisecond)},

		// ETHUSDT trades
		{Exchange: "binance", Symbol: "ETHUSDT", TradeID: 2001, Price: 3000.0, Quantity: 5.0, EventTime: baseTime.Add(1 * time.Millisecond)},
		{Exchange: "binance", Symbol: "ETHUSDT", TradeID: 2002, Price: 3001.0, Quantity: 3.0, EventTime: baseTime.Add(3 * time.Millisecond)},
		{Exchange: "binance", Symbol: "ETHUSDT", TradeID: 2003, Price: 3002.0, Quantity: 2.0, EventTime: baseTime.Add(5 * time.Millisecond)},

		// SOLUSDT trades
		{Exchange: "binance", Symbol: "SOLUSDT", TradeID: 3001, Price: 100.0, Quantity: 10.0, EventTime: baseTime.Add(2 * time.Millisecond)},
	}

	result := consumer.deduplicateBatch(input)

	// No duplicates expected - all trades should remain
	require.Equal(t, 9, len(result), "Should have 9 unique trades (no dedup in normal case)")

	// Verify trade IDs are preserved
	tradeIDs := make(map[int64]bool)
	for _, trade := range result {
		tradeIDs[trade.TradeID] = true
	}
	assert.Equal(t, 9, len(tradeIDs), "All trade IDs should be unique")
}

func TestWebSocketTradeConsumer_DeduplicateBatch_WithDuplicates(t *testing.T) {
	consumer := &WebSocketTradeConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Simulate edge case: same trade ID appears multiple times (network issues, etc.)
	input := []*market_data.Trade{
		// BTCUSDT - normal trades
		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1001, Price: 50000.0, EventTime: baseTime.Add(1 * time.Second)},
		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1002, Price: 50010.0, EventTime: baseTime.Add(2 * time.Second)},

		// Duplicate trade 1001 (earlier event_time - should be discarded)
		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1001, Price: 49990.0, EventTime: baseTime.Add(500 * time.Millisecond)},

		// Duplicate trade 1002 (later event_time - should be kept)
		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1002, Price: 50020.0, EventTime: baseTime.Add(3 * time.Second)},

		{Exchange: "binance", Symbol: "BTCUSDT", TradeID: 1003, Price: 50030.0, EventTime: baseTime.Add(4 * time.Second)},
	}

	result := consumer.deduplicateBatch(input)

	require.Equal(t, 3, len(result), "Should deduplicate to 3 unique trades")

	// Verify the latest versions are kept
	priceMap := make(map[int64]float64)
	for _, trade := range result {
		priceMap[trade.TradeID] = trade.Price
	}

	assert.Equal(t, 50000.0, priceMap[int64(1001)], "Trade 1001 should have first price (latest event_time)")
	assert.Equal(t, 50020.0, priceMap[int64(1002)], "Trade 1002 should have latest price (50020)")
	assert.Equal(t, 50030.0, priceMap[int64(1003)], "Trade 1003 should have price (50030)")
}

func TestWebSocketTradeConsumer_DeduplicateBatch_EmptyBatch(t *testing.T) {
	consumer := &WebSocketTradeConsumer{}

	result := consumer.deduplicateBatch([]*market_data.Trade{})
	assert.Empty(t, result, "Empty batch should return empty result")
}

func TestWebSocketTradeConsumer_DeduplicateBatch_PreservesAllFields(t *testing.T) {
	consumer := &WebSocketTradeConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.Trade{
		{
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			MarketType:   "futures",
			Timestamp:    baseTime,
			TradeID:      1001,
			AggTradeID:   5001,
			Price:        50000.0,
			Quantity:     1.5,
			FirstTradeID: 1000,
			LastTradeID:  1001,
			IsBuyerMaker: false,
			EventTime:    baseTime.Add(30 * time.Second),
		},
		{
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			TradeID:      1001, // Duplicate
			Price:        50075.0,
			Quantity:     2.0,
			IsBuyerMaker: true,
			EventTime:    baseTime.Add(45 * time.Second), // Later event
		},
	}

	result := consumer.deduplicateBatch(input)

	require.Len(t, result, 1, "Should deduplicate to 1 trade")

	trade := result[0]
	assert.Equal(t, "binance", trade.Exchange)
	assert.Equal(t, "BTCUSDT", trade.Symbol)
	assert.Equal(t, int64(1001), trade.TradeID)
	assert.Equal(t, 50075.0, trade.Price, "Should have latest price")
	assert.Equal(t, 2.0, trade.Quantity, "Should have latest quantity")
	assert.True(t, trade.IsBuyerMaker, "Should have latest buyer maker flag")
	assert.Equal(t, baseTime.Add(45*time.Second), trade.EventTime, "Should have latest event time")
}

func TestWebSocketTradeConsumer_DeduplicateBatch_DifferentExchanges(t *testing.T) {
	consumer := &WebSocketTradeConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.Trade{
		{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			TradeID:   1001,
			Price:     50000.0,
			EventTime: baseTime.Add(10 * time.Second),
		},
		{
			Exchange:  "bybit",
			Symbol:    "BTCUSDT",
			TradeID:   1001, // Same trade ID but different exchange
			Price:     50010.0,
			EventTime: baseTime.Add(10 * time.Second),
		},
	}

	result := consumer.deduplicateBatch(input)

	assert.Equal(t, 2, len(result), "Different exchanges should not deduplicate")
}

func TestWebSocketTradeConsumer_DeduplicateBatch_BuyerMakerFlag(t *testing.T) {
	consumer := &WebSocketTradeConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	input := []*market_data.Trade{
		{
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			TradeID:      1001,
			Price:        50000.0,
			IsBuyerMaker: true, // Sell trade
			EventTime:    baseTime.Add(10 * time.Second),
		},
		{
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			TradeID:      1002,
			Price:        50010.0,
			IsBuyerMaker: false, // Buy trade
			EventTime:    baseTime.Add(11 * time.Second),
		},
	}

	result := consumer.deduplicateBatch(input)

	require.Len(t, result, 2, "Different trade IDs - no dedup")
	
	// Verify Side() method works
	assert.Equal(t, "sell", result[0].Side(), "Trade 1001 should be sell (buyer maker)")
	assert.Equal(t, "buy", result[1].Side(), "Trade 1002 should be buy (seller maker)")
}

func TestWebSocketTradeConsumer_DeduplicateBatch_Performance(t *testing.T) {
	consumer := &WebSocketTradeConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Create large batch simulating high-frequency trading
	// 5 symbols × 200 trades each = 1000 events (normally no duplicates)
	// Add some duplicates for realism
	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "ADAUSDT"}
	tradesPerSymbol := 200

	input := make([]*market_data.Trade, 0, len(symbols)*tradesPerSymbol+50)

	tradeIDBase := int64(1000)
	for i, symbol := range symbols {
		for j := 0; j < tradesPerSymbol; j++ {
			tradeID := tradeIDBase + int64(i*tradesPerSymbol+j)
			input = append(input, &market_data.Trade{
				Exchange:     "binance",
				Symbol:       symbol,
				TradeID:      tradeID,
				Price:        50000.0 + float64(j),
				Quantity:     1.0,
				IsBuyerMaker: j%2 == 0,
				EventTime:    baseTime.Add(time.Duration(j) * time.Millisecond),
			})
		}
	}

	// Add some duplicates (10 random duplicate trades)
	for i := 0; i < 10; i++ {
		input = append(input, &market_data.Trade{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			TradeID:   1000 + int64(i*20), // Duplicate some trades
			Price:     50100.0 + float64(i),
			EventTime: baseTime.Add(time.Duration(500+i) * time.Millisecond), // Later event
		})
	}

	require.Len(t, input, 1010, "Should have 1010 input events")

	start := time.Now()
	result := consumer.deduplicateBatch(input)
	duration := time.Since(start)

	expectedUnique := len(symbols)*tradesPerSymbol // 1000 (10 duplicates removed)
	assert.Equal(t, expectedUnique, len(result), "Should deduplicate 1010 → 1000 events")
	assert.Less(t, duration.Milliseconds(), int64(20), "Deduplication should be fast (<20ms)")

	t.Logf("✅ Deduplicated %d → %d events in %v", len(input), len(result), duration)
}

func TestWebSocketTradeConsumer_DeduplicateBatch_AggregatedTrades(t *testing.T) {
	consumer := &WebSocketTradeConsumer{}
	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	// Test aggregated trades (multiple raw trades aggregated into one)
	input := []*market_data.Trade{
		{
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			TradeID:      1001,
			AggTradeID:   5001,
			Price:        50000.0,
			Quantity:     10.5,
			FirstTradeID: 1000,
			LastTradeID:  1010,
			IsBuyerMaker: false,
			EventTime:    baseTime.Add(1 * time.Second),
		},
		{
			Exchange:     "binance",
			Symbol:       "BTCUSDT",
			TradeID:      1002,
			AggTradeID:   5002,
			Price:        50010.0,
			Quantity:     5.3,
			FirstTradeID: 1011,
			LastTradeID:  1015,
			IsBuyerMaker: true,
			EventTime:    baseTime.Add(2 * time.Second),
		},
	}

	result := consumer.deduplicateBatch(input)

	require.Len(t, result, 2, "Should keep both aggregated trades")
	
	// Verify both agg trade IDs are present (order not guaranteed due to map iteration)
	aggTradeIDs := make(map[int64]bool)
	for _, trade := range result {
		aggTradeIDs[trade.AggTradeID] = true
	}
	assert.True(t, aggTradeIDs[int64(5001)], "Should preserve agg trade ID 5001")
	assert.True(t, aggTradeIDs[int64(5002)], "Should preserve agg trade ID 5002")
}

