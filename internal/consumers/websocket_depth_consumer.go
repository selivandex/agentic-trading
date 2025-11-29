package consumers

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/logger"
)

const (
	depthBatchSize     = 50 // Smaller batch for depth data (larger payloads)
	depthFlushInterval = 3 * time.Second
	depthStatsInterval = 1 * time.Minute
)

// WebSocketDepthConsumer consumes depth/orderbook events from Kafka and writes to ClickHouse
type WebSocketDepthConsumer struct {
	*WebSocketBaseConsumer[*eventspb.WebSocketDepthEvent, market_data.OrderBookSnapshot]
	repository market_data.Repository
}

// depthHandler implements EventHandler interface
type depthHandler struct {
	repository market_data.Repository
}

// NewWebSocketDepthConsumer creates a new depth consumer
func NewWebSocketDepthConsumer(
	consumer *kafkaadapter.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketDepthConsumer {
	handler := &depthHandler{repository: repository}

	baseConsumer := NewWebSocketBaseConsumer[*eventspb.WebSocketDepthEvent, market_data.OrderBookSnapshot](
		consumer,
		repository,
		log,
		BatchProcessingConfig{
			ConsumerName:  "WebSocket Depth Consumer",
			BatchSize:     depthBatchSize,
			FlushInterval: depthFlushInterval,
			StatsInterval: depthStatsInterval,
		},
		handler,
	)

	return &WebSocketDepthConsumer{
		WebSocketBaseConsumer: baseConsumer,
		repository:            repository,
	}
}

// UnmarshalEvent implements EventHandler interface
func (h *depthHandler) UnmarshalEvent(data []byte) (*eventspb.WebSocketDepthEvent, error) {
	var event eventspb.WebSocketDepthEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// ConvertToDomain implements EventHandler interface
func (h *depthHandler) ConvertToDomain(event *eventspb.WebSocketDepthEvent) market_data.OrderBookSnapshot {
	// Convert price levels to JSON
	type PriceLevel struct {
		Price    string `json:"p"`
		Quantity string `json:"q"`
	}

	bids := make([]PriceLevel, len(event.Bids))
	var bidDepth float64
	for i, bid := range event.Bids {
		bids[i] = PriceLevel{
			Price:    bid.Price,
			Quantity: bid.Quantity,
		}
		qty, _ := strconv.ParseFloat(bid.Quantity, 64)
		bidDepth += qty
	}

	asks := make([]PriceLevel, len(event.Asks))
	var askDepth float64
	for i, ask := range event.Asks {
		asks[i] = PriceLevel{
			Price:    ask.Price,
			Quantity: ask.Quantity,
		}
		qty, _ := strconv.ParseFloat(ask.Quantity, 64)
		askDepth += qty
	}

	// Serialize to JSON
	bidsJSON, _ := json.Marshal(bids)
	asksJSON, _ := json.Marshal(asks)

	// Use market_type from event, fallback to "futures" for backward compatibility
	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	return market_data.OrderBookSnapshot{
		Exchange:   event.Exchange,
		Symbol:     event.Symbol,
		MarketType: marketType,
		Timestamp:  event.EventTime.AsTime(),
		Bids:       string(bidsJSON),
		Asks:       string(asksJSON),
		BidDepth:   bidDepth,
		AskDepth:   askDepth,
		EventTime:  event.EventTime.AsTime(),
	}
}

// GetDeduplicationKey implements EventHandler interface
func (h *depthHandler) GetDeduplicationKey(item market_data.OrderBookSnapshot) string {
	return item.Exchange + "_" + item.Symbol + "_" + strconv.FormatInt(item.Timestamp.Unix(), 10)
}

// GetEventTime implements EventHandler interface
func (h *depthHandler) GetEventTime(item market_data.OrderBookSnapshot) time.Time {
	return item.EventTime
}

// InsertBatch implements EventHandler interface
func (h *depthHandler) InsertBatch(ctx context.Context, batch []market_data.OrderBookSnapshot) error {
	return h.repository.InsertOrderBook(ctx, batch)
}

// GetCustomStats implements EventHandler interface
func (h *depthHandler) GetCustomStats() map[string]interface{} {
	return map[string]interface{}{} // No custom stats for depth
}
