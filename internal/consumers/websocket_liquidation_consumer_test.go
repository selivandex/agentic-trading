package consumers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventspb "prometheus/internal/events/proto"
)

func TestLiquidationHandler_ConvertToDomain(t *testing.T) {
	handler := &liquidationHandler{}

	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	event := &eventspb.WebSocketLiquidationEvent{
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		MarketType: "futures",
		Side:       "long",
		OrderType:  "MARKET",
		Price:      "50000.50",
		Quantity:   "1.5",
		Value:      "75000.75",
		EventTime:  timestamppb.New(baseTime),
	}

	liquidation := handler.ConvertToDomain(event)

	assert.Equal(t, "binance", liquidation.Exchange)
	assert.Equal(t, "BTCUSDT", liquidation.Symbol)
	assert.Equal(t, "futures", liquidation.MarketType)
	assert.Equal(t, "long", liquidation.Side)
	assert.Equal(t, "MARKET", liquidation.OrderType)
	assert.InDelta(t, 50000.50, liquidation.Price, 0.01)
	assert.InDelta(t, 1.5, liquidation.Quantity, 0.01)
	assert.InDelta(t, 75000.75, liquidation.Value, 0.01)
	assert.Equal(t, baseTime, liquidation.EventTime)
}

func TestLiquidationHandler_ConvertToDomain_InvalidNumbers(t *testing.T) {
	handler := &liquidationHandler{}

	baseTime := time.Now()

	// Test with invalid numbers (should handle gracefully with 0 values)
	event := &eventspb.WebSocketLiquidationEvent{
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		MarketType: "futures",
		Side:       "short",
		OrderType:  "MARKET",
		Price:      "invalid",
		Quantity:   "not-a-number",
		Value:      "bad-value",
		EventTime:  timestamppb.New(baseTime),
	}

	liquidation := handler.ConvertToDomain(event)

	// Invalid numbers should parse as 0
	assert.Equal(t, 0.0, liquidation.Price)
	assert.Equal(t, 0.0, liquidation.Quantity)
	assert.Equal(t, 0.0, liquidation.Value)

	// Other fields should still be correct
	assert.Equal(t, "binance", liquidation.Exchange)
	assert.Equal(t, "BTCUSDT", liquidation.Symbol)
	assert.Equal(t, "short", liquidation.Side)
}

func TestLiquidationHandler_GetDeduplicationKey(t *testing.T) {
	handler := &liquidationHandler{}

	event := &eventspb.WebSocketLiquidationEvent{
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		MarketType: "futures",
		Side:       "long",
		OrderType:  "MARKET",
		Price:      "50000.0",
		Quantity:   "1.0",
		Value:      "50000.0",
		EventTime:  timestamppb.New(time.Now()),
	}

	liquidation := handler.ConvertToDomain(event)

	// Liquidations don't need deduplication (no unique ID)
	key := handler.GetDeduplicationKey(liquidation)
	assert.Empty(t, key, "Liquidations should not have deduplication key")
}
