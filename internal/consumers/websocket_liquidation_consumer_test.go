package consumers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"prometheus/internal/domain/market_data"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/logger"
)

func TestWebSocketLiquidationConsumer_convertProtobufToLiquidation(t *testing.T) {
	// Initialize logger for tests
	_ = logger.Init("info", "test")
	
	consumer := &WebSocketLiquidationConsumer{
		log: logger.Get(),
	}

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

	liquidation := consumer.convertProtobufToLiquidation(event)

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

func TestWebSocketLiquidationConsumer_addToBatch(t *testing.T) {
	consumer := NewWebSocketLiquidationConsumer(
		nil, // No Kafka consumer needed for this test
		nil, // No repository needed for this test
		logger.NewLogger(false),
	)

	baseTime := time.Now()

	liquidation1 := &market_data.Liquidation{
		Exchange:   "binance",
		Symbol:     "BTCUSDT",
		MarketType: "futures",
		Timestamp:  baseTime,
		Side:       "long",
		OrderType:  "MARKET",
		Price:      50000.0,
		Quantity:   1.0,
		Value:      50000.0,
		EventTime:  baseTime,
	}

	liquidation2 := &market_data.Liquidation{
		Exchange:   "binance",
		Symbol:     "ETHUSDT",
		MarketType: "futures",
		Timestamp:  baseTime,
		Side:       "short",
		OrderType:  "LIMIT",
		Price:      3000.0,
		Quantity:   10.0,
		Value:      30000.0,
		EventTime:  baseTime,
	}

	// Add liquidations to batch
	consumer.addToBatch(liquidation1)
	consumer.addToBatch(liquidation2)

	// Verify batch size
	consumer.mu.Lock()
	batchSize := len(consumer.batch)
	consumer.mu.Unlock()

	assert.Equal(t, 2, batchSize)
}

func TestWebSocketLiquidationConsumer_flushEmptyBatch(t *testing.T) {
	consumer := NewWebSocketLiquidationConsumer(
		nil,
		nil,
		logger.NewLogger(false),
	)

	ctx := context.Background()

	// Flushing empty batch should not error
	err := consumer.flushBatch(ctx)
	require.NoError(t, err)
}

func TestWebSocketLiquidationConsumer_incrementStat(t *testing.T) {
	consumer := NewWebSocketLiquidationConsumer(
		nil,
		nil,
		logger.NewLogger(false),
	)

	initialValue := consumer.totalReceived

	consumer.incrementStat(&consumer.totalReceived)
	consumer.incrementStat(&consumer.totalReceived)
	consumer.incrementStat(&consumer.totalReceived)

	assert.Equal(t, initialValue+3, consumer.totalReceived)
}

func TestWebSocketLiquidationConsumer_convertProtobufToLiquidation_InvalidNumbers(t *testing.T) {
	consumer := &WebSocketLiquidationConsumer{
		log: logger.NewLogger(false),
	}

	baseTime := time.Now()

	// Test with invalid numbers (should handle gracefully)
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

	liquidation := consumer.convertProtobufToLiquidation(event)

	// Invalid numbers should parse as 0
	assert.Equal(t, 0.0, liquidation.Price)
	assert.Equal(t, 0.0, liquidation.Quantity)
	assert.Equal(t, 0.0, liquidation.Value)

	// Other fields should still be correct
	assert.Equal(t, "binance", liquidation.Exchange)
	assert.Equal(t, "BTCUSDT", liquidation.Symbol)
	assert.Equal(t, "short", liquidation.Side)
}

