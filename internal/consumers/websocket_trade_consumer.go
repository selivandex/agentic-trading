package consumers

import (
	"context"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/logger"
)

const (
	tradeBatchSize     = 500 // Larger batch - trades come very frequently
	tradeFlushInterval = 3 * time.Second
	tradeStatsInterval = 1 * time.Minute
)

// WebSocketTradeConsumer consumes trade events from Kafka and writes to ClickHouse
type WebSocketTradeConsumer struct {
	*WebSocketBaseConsumer[*eventspb.WebSocketTradeEvent, market_data.Trade]
	repository market_data.Repository
}

// tradeHandler implements EventHandler interface
type tradeHandler struct {
	repository market_data.Repository
}

// NewWebSocketTradeConsumer creates a new trade consumer
func NewWebSocketTradeConsumer(
	consumer *kafkaadapter.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketTradeConsumer {
	handler := &tradeHandler{repository: repository}

	baseConsumer := NewWebSocketBaseConsumer[*eventspb.WebSocketTradeEvent, market_data.Trade](
		consumer,
		repository,
		log,
		BatchProcessingConfig{
			ConsumerName:  "WebSocket Trade Consumer",
			BatchSize:     tradeBatchSize,
			FlushInterval: tradeFlushInterval,
			StatsInterval: tradeStatsInterval,
		},
		handler,
	)

	return &WebSocketTradeConsumer{
		WebSocketBaseConsumer: baseConsumer,
		repository:            repository,
	}
}

// UnmarshalEvent implements EventHandler interface
func (h *tradeHandler) UnmarshalEvent(data []byte) (*eventspb.WebSocketTradeEvent, error) {
	var event eventspb.WebSocketTradeEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ConvertToDomain implements EventHandler interface
func (h *tradeHandler) ConvertToDomain(event *eventspb.WebSocketTradeEvent) market_data.Trade {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	// Use market_type from event, fallback to "futures" for backward compatibility
	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	return market_data.Trade{
		Exchange:     event.Exchange,
		Symbol:       event.Symbol,
		MarketType:   marketType,
		Timestamp:    event.TradeTime.AsTime(),
		TradeID:      event.TradeId,
		AggTradeID:   event.TradeId, // For aggTrade this is the agg ID
		Price:        parseFloat(event.Price),
		Quantity:     parseFloat(event.Quantity),
		FirstTradeID: event.TradeId, // In aggTrade, would be different
		LastTradeID:  event.TradeId,
		IsBuyerMaker: event.IsBuyerMaker,
		EventTime:    event.EventTime.AsTime(),
	}
}

// GetDeduplicationKey implements EventHandler interface
func (h *tradeHandler) GetDeduplicationKey(item market_data.Trade) string {
	return item.Exchange + "_" + item.Symbol + "_" + strconv.FormatInt(item.TradeID, 10)
}

// GetEventTime implements EventHandler interface
func (h *tradeHandler) GetEventTime(item market_data.Trade) time.Time {
	return item.EventTime
}

// InsertBatch implements EventHandler interface
func (h *tradeHandler) InsertBatch(ctx context.Context, batch []market_data.Trade) error {
	return h.repository.InsertTrades(ctx, batch)
}

// GetCustomStats implements EventHandler interface
func (h *tradeHandler) GetCustomStats() map[string]interface{} {
	return map[string]interface{}{} // No custom stats for trades
}
