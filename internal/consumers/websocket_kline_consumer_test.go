package consumers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventspb "prometheus/internal/events/proto"
)

// TestKlineHandler_ConvertToDomain tests the conversion from protobuf to domain model
func TestKlineHandler_ConvertToDomain(t *testing.T) {
	handler := &klineHandler{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)
	closeTime := baseTime.Add(1 * time.Minute)

	tests := []struct {
		name  string
		event *eventspb.WebSocketKlineEvent
		check func(t *testing.T, handler *klineHandler)
	}{
		{
			name: "final_candle",
			event: &eventspb.WebSocketKlineEvent{
				Exchange:    "binance",
				Symbol:      "BTCUSDT",
				MarketType:  "futures",
				Interval:    "1m",
				OpenTime:    timestamppb.New(baseTime),
				CloseTime:   timestamppb.New(closeTime),
				Open:        "50000.0",
				High:        "50100.0",
				Low:         "49900.0",
				Close:       "50050.0",
				Volume:      "100.5",
				QuoteVolume: "5025000.0",
				TradeCount:  1000,
				IsFinal:     true,
				EventTime:   timestamppb.New(baseTime.Add(10 * time.Second)),
			},
			check: func(t *testing.T, handler *klineHandler) {
				// Check that final candle incremented totalFinal counter
				handler.statsMu.Lock()
				defer handler.statsMu.Unlock()
				assert.Equal(t, int64(1), handler.totalFinal)
				assert.Equal(t, int64(0), handler.totalNonFinal)
			},
		},
		{
			name: "non_final_candle",
			event: &eventspb.WebSocketKlineEvent{
				Exchange:    "binance",
				Symbol:      "ETHUSDT",
				MarketType:  "futures",
				Interval:    "5m",
				OpenTime:    timestamppb.New(baseTime),
				CloseTime:   timestamppb.New(closeTime.Add(4 * time.Minute)),
				Open:        "3000.0",
				High:        "3010.0",
				Low:         "2990.0",
				Close:       "3005.0",
				Volume:      "50.25",
				QuoteVolume: "150750.0",
				TradeCount:  500,
				IsFinal:     false,
				EventTime:   timestamppb.New(baseTime.Add(30 * time.Second)),
			},
			check: func(t *testing.T, handler *klineHandler) {
				// Check that non-final candle incremented totalNonFinal counter
				// Note: we start with 1 final from previous test
				handler.statsMu.Lock()
				defer handler.statsMu.Unlock()
				assert.Equal(t, int64(1), handler.totalFinal)
				assert.Equal(t, int64(1), handler.totalNonFinal)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ConvertToDomain(tt.event)

			// Verify basic conversion
			assert.Equal(t, tt.event.Exchange, result.Exchange)
			assert.Equal(t, tt.event.Symbol, result.Symbol)
			assert.Equal(t, tt.event.Interval, result.Timeframe)
			assert.Equal(t, tt.event.IsFinal, result.IsClosed)
			assert.Equal(t, tt.event.OpenTime.AsTime(), result.OpenTime)
			assert.Equal(t, tt.event.CloseTime.AsTime(), result.CloseTime)
			assert.Equal(t, tt.event.EventTime.AsTime(), result.EventTime)

			// Run custom check
			if tt.check != nil {
				tt.check(t, handler)
			}
		})
	}
}

// TestKlineHandler_GetDeduplicationKey tests deduplication key generation
func TestKlineHandler_GetDeduplicationKey(t *testing.T) {
	handler := &klineHandler{}

	baseTime := time.Date(2025, 11, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		ohlcv1      *eventspb.WebSocketKlineEvent
		ohlcv2      *eventspb.WebSocketKlineEvent
		shouldMatch bool
	}{
		{
			name: "same_key_different_close",
			ohlcv1: &eventspb.WebSocketKlineEvent{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				OpenTime: timestamppb.New(baseTime),
				Close:    "50000.0",
			},
			ohlcv2: &eventspb.WebSocketKlineEvent{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				OpenTime: timestamppb.New(baseTime),
				Close:    "50100.0", // Different close - should deduplicate
			},
			shouldMatch: true,
		},
		{
			name: "different_interval",
			ohlcv1: &eventspb.WebSocketKlineEvent{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				OpenTime: timestamppb.New(baseTime),
			},
			ohlcv2: &eventspb.WebSocketKlineEvent{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "5m", // Different interval
				OpenTime: timestamppb.New(baseTime),
			},
			shouldMatch: false,
		},
		{
			name: "different_open_time",
			ohlcv1: &eventspb.WebSocketKlineEvent{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				OpenTime: timestamppb.New(baseTime),
			},
			ohlcv2: &eventspb.WebSocketKlineEvent{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				OpenTime: timestamppb.New(baseTime.Add(1 * time.Minute)), // Different open time
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain1 := handler.ConvertToDomain(tt.ohlcv1)
			domain2 := handler.ConvertToDomain(tt.ohlcv2)

			key1 := handler.GetDeduplicationKey(domain1)
			key2 := handler.GetDeduplicationKey(domain2)

			if tt.shouldMatch {
				assert.Equal(t, key1, key2, "Keys should match for deduplication")
			} else {
				assert.NotEqual(t, key1, key2, "Keys should not match")
			}
		})
	}
}
