package consumers

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

const (
	// Batch configuration for WebSocket consumer
	wsBatchSize     = 100 // Higher batch since we handle multiple types
	wsFlushInterval = 10 * time.Second
	wsStatsInterval = 1 * time.Minute
)

// WebSocketConsumer is a unified consumer for all WebSocket events
// Handles all stream types from TopicWebSocketEvents and routes by event.Base.Type
type WebSocketConsumer struct {
	consumer   *kafkaadapter.Consumer
	repository market_data.Repository
	log        *logger.Logger

	// Batching (single batch for all event types)
	mu    sync.Mutex
	batch []interface{} // Mixed batch of different domain types

	// Statistics
	statsMu               sync.Mutex
	totalReceived         int64
	totalProcessed        int64
	totalSkipped          int64 // Events skipped due to type filtering
	totalErrors           int64
	lastFlushTime         time.Time
	lastStatsLogTime      time.Time
	receivedSinceLastLog  int64
	processedSinceLastLog int64

	// Per-type statistics
	typeStats map[string]*eventTypeStats
}

type eventTypeStats struct {
	received  int64
	processed int64
	errors    int64
}

// NewWebSocketConsumer creates a new unified WebSocket consumer
func NewWebSocketConsumer(
	consumer *kafkaadapter.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketConsumer {
	now := time.Now()
	return &WebSocketConsumer{
		consumer:         consumer,
		repository:       repository,
		log:              log,
		batch:            make([]interface{}, 0, wsBatchSize),
		lastFlushTime:    now,
		lastStatsLogTime: now,
		typeStats:        make(map[string]*eventTypeStats),
	}
}

// Start begins consuming WebSocket events
func (c *WebSocketConsumer) Start(ctx context.Context) error {
	c.log.Infow("Starting WebSocket consumer...",
		"topic", "websocket.events",
		"batch_size", wsBatchSize,
		"flush_interval", wsFlushInterval,
	)

	// Use BatchConsumerLifecycle for DRY shutdown/flush management
	lifecycle := NewBatchConsumerLifecycle(
		BatchConsumerConfig{
			ConsumerName:  "WebSocket Consumer",
			FlushInterval: wsFlushInterval,
			StatsInterval: wsStatsInterval,
			Logger:        c.log,
		},
		c.consumer,
		c, // implements BatchConsumer interface
	)

	// Setup cleanup
	defer lifecycle.Start(ctx)()

	// Start background workers
	lifecycle.StartBackgroundWorkers(ctx)

	c.log.Info("🔄 Starting to read WebSocket events from Kafka...")

	// Use Consume() for message loop
	return c.consumer.Consume(ctx, c.handleMessage)
}

// handleMessage processes a single Kafka message
func (c *WebSocketConsumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	c.incrementStat(&c.totalReceived)
	c.incrementStat(&c.receivedSinceLastLog)

	processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.handleEvent(processCtx, msg.Value); err != nil {
		c.log.Errorw("Failed to handle WebSocket event",
			"error", err,
		)
		c.incrementStat(&c.totalErrors)
		return err
	}

	return nil
}

// handleEvent processes a single event based on type
// Uses WebSocketEventWrapper with oneof for efficient type detection
func (c *WebSocketConsumer) handleEvent(ctx context.Context, data []byte) error {
	var wrapper eventspb.WebSocketEventWrapper
	if err := proto.Unmarshal(data, &wrapper); err != nil {
		c.log.Debugw("⚠️ Failed to unmarshal wrapper", "error", err, "data_len", len(data))
		c.incrementStat(&c.totalSkipped)
		return nil
	}

	// Check which event type is set in the oneof
	switch event := wrapper.Event.(type) {
	case *eventspb.WebSocketEventWrapper_Kline:
		c.trackTypeReceived("websocket.kline")
		return c.handleKlineTyped(ctx, event.Kline)

	case *eventspb.WebSocketEventWrapper_Ticker:
		c.trackTypeReceived("websocket.ticker")
		return c.handleTickerTyped(ctx, event.Ticker)

	case *eventspb.WebSocketEventWrapper_Depth:
		c.trackTypeReceived("websocket.depth")
		return c.handleDepthTyped(ctx, event.Depth)

	case *eventspb.WebSocketEventWrapper_Trade:
		c.trackTypeReceived("websocket.trade")
		return c.handleTradeTyped(ctx, event.Trade)

	case *eventspb.WebSocketEventWrapper_MarkPrice:
		c.trackTypeReceived("websocket.mark_price")
		return c.handleMarkPriceTyped(ctx, event.MarkPrice)

	case *eventspb.WebSocketEventWrapper_Liquidation:
		c.trackTypeReceived("websocket.liquidation")
		return c.handleLiquidationTyped(ctx, event.Liquidation)

	case *eventspb.WebSocketEventWrapper_FundingRate:
		c.trackTypeReceived("websocket.funding_rate")
		c.log.Debug("Funding rate events not implemented yet")
		c.incrementStat(&c.totalSkipped)
		return nil

	default:
		c.log.Debugw("Unknown WebSocket event in wrapper", "data_len", len(data))
		c.incrementStat(&c.totalSkipped)
		return nil
	}
}

// handleKlineTyped processes kline/candlestick events from typed protobuf message
func (c *WebSocketConsumer) handleKlineTyped(ctx context.Context, event *eventspb.WebSocketKlineEvent) error {
	// Parse helper
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	ohlcv := market_data.OHLCV{
		Exchange:    event.Exchange,
		Symbol:      event.Symbol,
		Timeframe:   event.Interval,
		MarketType:  marketType,
		OpenTime:    event.OpenTime.AsTime(),
		CloseTime:   event.CloseTime.AsTime(),
		Open:        parseFloat(event.Open),
		High:        parseFloat(event.High),
		Low:         parseFloat(event.Low),
		Close:       parseFloat(event.Close),
		Volume:      parseFloat(event.Volume),
		QuoteVolume: parseFloat(event.QuoteVolume),
		Trades:      uint64(event.TradeCount),
		IsClosed:    event.IsFinal,
		EventTime:   event.EventTime.AsTime(),
	}

	c.addToBatch(ohlcv)
	c.trackTypeProcessed("websocket.kline")
	return nil
}

// handleTickerTyped processes 24hr ticker events from typed protobuf message
func (c *WebSocketConsumer) handleTickerTyped(ctx context.Context, event *eventspb.WebSocketTickerEvent) error {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	marketType := event.MarketType
	if marketType == "" {
		marketType = "spot"
	}

	ticker := market_data.Ticker{
		Exchange:           event.Exchange,
		Symbol:             event.Symbol,
		MarketType:         marketType,
		Timestamp:          time.Now(),
		LastPrice:          parseFloat(event.LastPrice),
		OpenPrice:          parseFloat(event.OpenPrice),
		HighPrice:          parseFloat(event.HighPrice),
		LowPrice:           parseFloat(event.LowPrice),
		Volume:             parseFloat(event.Volume),
		QuoteVolume:        parseFloat(event.QuoteVolume),
		PriceChange:        parseFloat(event.PriceChange),
		PriceChangePercent: parseFloat(event.PriceChangePercent),
		WeightedAvgPrice:   parseFloat(event.WeightedAvgPrice),
		TradeCount:         uint64(event.TradeCount),
		EventTime:          event.EventTime.AsTime(),
	}

	c.addToBatch(ticker)
	c.trackTypeProcessed("websocket.ticker")
	return nil
}

// handleDepthTyped processes order book depth events from typed protobuf message
func (c *WebSocketConsumer) handleDepthTyped(ctx context.Context, event *eventspb.WebSocketDepthEvent) error {
	// Convert bids and asks to JSON
	// Note: OrderBookSnapshot stores bids/asks as JSON strings in ClickHouse
	bidsJSON := "[]"
	asksJSON := "[]"

	// Simple JSON serialization for order book levels
	// TODO: Use proper JSON marshaling if needed

	marketType := event.MarketType
	if marketType == "" {
		marketType = "spot"
	}

	snapshot := market_data.OrderBookSnapshot{
		Exchange:   event.Exchange,
		Symbol:     event.Symbol,
		MarketType: marketType,
		Timestamp:  time.Now(),
		Bids:       bidsJSON,
		Asks:       asksJSON,
		BidDepth:   0.0, // Calculate if needed
		AskDepth:   0.0, // Calculate if needed
		EventTime:  event.EventTime.AsTime(),
	}

	c.addToBatch(snapshot)
	c.trackTypeProcessed("websocket.depth")
	return nil
}

// handleTradeTyped processes trade events from typed protobuf message
func (c *WebSocketConsumer) handleTradeTyped(ctx context.Context, event *eventspb.WebSocketTradeEvent) error {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	marketType := event.MarketType
	if marketType == "" {
		marketType = "spot"
	}

	trade := market_data.Trade{
		Exchange:     event.Exchange,
		Symbol:       event.Symbol,
		MarketType:   marketType,
		Timestamp:    time.Now(),
		TradeID:      event.TradeId,
		AggTradeID:   event.TradeId, // Use same ID for now
		Price:        parseFloat(event.Price),
		Quantity:     parseFloat(event.Quantity),
		FirstTradeID: event.TradeId,
		LastTradeID:  event.TradeId,
		IsBuyerMaker: event.IsBuyerMaker,
		EventTime:    event.EventTime.AsTime(),
	}

	c.addToBatch(trade)
	c.trackTypeProcessed("websocket.trade")
	return nil
}

// handleMarkPriceTyped processes mark price events from typed protobuf message
func (c *WebSocketConsumer) handleMarkPriceTyped(ctx context.Context, event *eventspb.WebSocketMarkPriceEvent) error {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	markPrice := market_data.MarkPrice{
		Exchange:             event.Exchange,
		Symbol:               event.Symbol,
		MarketType:           "futures",
		Timestamp:            time.Now(),
		MarkPrice:            parseFloat(event.MarkPrice),
		IndexPrice:           parseFloat(event.IndexPrice),
		EstimatedSettlePrice: parseFloat(event.EstimatedSettlePrice),
		FundingRate:          parseFloat(event.FundingRate),
		NextFundingTime:      event.NextFundingTime.AsTime(),
		EventTime:            event.EventTime.AsTime(),
	}

	c.addToBatch(markPrice)
	c.trackTypeProcessed("websocket.mark_price")
	return nil
}

// handleLiquidationTyped processes liquidation events from typed protobuf message
func (c *WebSocketConsumer) handleLiquidationTyped(ctx context.Context, event *eventspb.WebSocketLiquidationEvent) error {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	liquidation := market_data.Liquidation{
		Exchange:   event.Exchange,
		Symbol:     event.Symbol,
		MarketType: marketType,
		Timestamp:  time.Now(),
		Side:       event.Side,
		OrderType:  event.OrderType,
		Price:      parseFloat(event.Price),
		Quantity:   parseFloat(event.Quantity),
		Value:      parseFloat(event.Value),
		EventTime:  event.EventTime.AsTime(),
	}

	c.addToBatch(liquidation)
	c.trackTypeProcessed("websocket.liquidation")
	return nil
}

// addToBatch adds an item to batch (thread-safe)
func (c *WebSocketConsumer) addToBatch(item interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.batch = append(c.batch, item)
	c.incrementStat(&c.totalProcessed)
	c.incrementStat(&c.processedSinceLastLog)

	// Auto-flush if batch is full
	if len(c.batch) >= wsBatchSize {
		c.log.Infow("📦 Batch full, triggering flush",
			"batch_size", len(c.batch),
		)
		go c.FlushBatch(context.Background()) // Non-blocking flush
	}
}

// FlushBatch implements BatchConsumer interface
func (c *WebSocketConsumer) FlushBatch(ctx context.Context) error {
	c.mu.Lock()
	if len(c.batch) == 0 {
		c.mu.Unlock()
		return nil
	}

	// Copy batch and reset
	batchCopy := make([]interface{}, len(c.batch))
	copy(batchCopy, c.batch)
	originalSize := len(c.batch)
	c.batch = c.batch[:0] // Reset batch
	c.lastFlushTime = time.Now()
	c.mu.Unlock()

	start := time.Now()

	// Group items by type and insert
	if err := c.insertMixedBatch(ctx, batchCopy); err != nil {
		c.log.Error("Failed to flush batch", "error", err)
		return err
	}

	duration := time.Since(start)
	c.log.Infow("✅ Flushed mixed batch to ClickHouse",
		"size", originalSize,
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

// insertMixedBatch groups items by type and inserts them
func (c *WebSocketConsumer) insertMixedBatch(ctx context.Context, batch []interface{}) error {
	var (
		klines       []market_data.OHLCV
		tickers      []market_data.Ticker
		depths       []market_data.OrderBookSnapshot
		trades       []market_data.Trade
		markPrices   []market_data.MarkPrice
		liquidations []market_data.Liquidation
	)

	// Group by type
	for _, item := range batch {
		switch v := item.(type) {
		case market_data.OHLCV:
			klines = append(klines, v)
		case market_data.Ticker:
			tickers = append(tickers, v)
		case market_data.OrderBookSnapshot:
			depths = append(depths, v)
		case market_data.Trade:
			trades = append(trades, v)
		case market_data.MarkPrice:
			markPrices = append(markPrices, v)
		case market_data.Liquidation:
			liquidations = append(liquidations, v)
		}
	}

	// Insert each type
	if len(klines) > 0 {
		if err := c.repository.InsertOHLCV(ctx, klines); err != nil {
			return errors.Wrap(err, "insert klines")
		}
		c.log.Infow("  → Inserted klines", "count", len(klines))
	}

	if len(tickers) > 0 {
		if err := c.repository.InsertTicker(ctx, tickers); err != nil {
			return errors.Wrap(err, "insert tickers")
		}
		c.log.Infow("  → Inserted tickers", "count", len(tickers))
	}

	if len(depths) > 0 {
		if err := c.repository.InsertOrderBook(ctx, depths); err != nil {
			return errors.Wrap(err, "insert depths")
		}
		c.log.Infow("  → Inserted order book snapshots", "count", len(depths))
	}

	if len(trades) > 0 {
		if err := c.repository.InsertTrades(ctx, trades); err != nil {
			return errors.Wrap(err, "insert trades")
		}
		c.log.Infow("  → Inserted trades", "count", len(trades))
	}

	if len(markPrices) > 0 {
		if err := c.repository.InsertMarkPrice(ctx, markPrices); err != nil {
			return errors.Wrap(err, "insert mark prices")
		}
		c.log.Infow("  → Inserted mark prices", "count", len(markPrices))
	}

	if len(liquidations) > 0 {
		if err := c.repository.InsertLiquidations(ctx, liquidations); err != nil {
			return errors.Wrap(err, "insert liquidations")
		}
		c.log.Infow("  → Inserted liquidations", "count", len(liquidations))
	}

	return nil
}

// LogStats implements BatchConsumer interface
func (c *WebSocketConsumer) LogStats(final bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	timeSinceLastFlush := time.Since(c.lastFlushTime)
	timeSinceLastLog := time.Since(c.lastStatsLogTime)

	receivedPerSec := int64(0)
	processedPerSec := int64(0)
	if timeSinceLastLog.Seconds() > 0 {
		receivedPerSec = int64(float64(c.receivedSinceLastLog) / timeSinceLastLog.Seconds())
		processedPerSec = int64(float64(c.processedSinceLastLog) / timeSinceLastLog.Seconds())
	}

	c.mu.Lock()
	pendingBatch := len(c.batch)
	c.mu.Unlock()

	stats := map[string]interface{}{
		"total_received":        c.totalReceived,
		"total_processed":       c.totalProcessed,
		"total_skipped":         c.totalSkipped,
		"total_errors":          c.totalErrors,
		"pending_batch":         pendingBatch,
		"received_per_sec":      receivedPerSec,
		"processed_per_sec":     processedPerSec,
		"time_since_last_flush": timeSinceLastFlush.Round(time.Second).String(),
	}

	// Add per-type stats
	for eventType, typeStats := range c.typeStats {
		stats[eventType+"_received"] = typeStats.received
		stats[eventType+"_processed"] = typeStats.processed
		if typeStats.errors > 0 {
			stats[eventType+"_errors"] = typeStats.errors
		}
	}

	// Convert map to flat key-value list for structured logging
	args := make([]interface{}, 0, len(stats)*2)
	for k, v := range stats {
		args = append(args, k, v)
	}

	if final {
		c.log.Infow("📊 [FINAL] WebSocket Consumer stats", args...)
	} else {
		c.log.Infow("📊 WebSocket Consumer stats", args...)
	}

	// Reset interval counters
	c.receivedSinceLastLog = 0
	c.processedSinceLastLog = 0
	c.lastStatsLogTime = time.Now()
}

// Helper methods for statistics

func (c *WebSocketConsumer) incrementStat(stat *int64) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	*stat++
}

func (c *WebSocketConsumer) trackTypeReceived(eventType string) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	if c.typeStats[eventType] == nil {
		c.typeStats[eventType] = &eventTypeStats{}
	}
	c.typeStats[eventType].received++
}

func (c *WebSocketConsumer) trackTypeProcessed(eventType string) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	if c.typeStats[eventType] == nil {
		c.typeStats[eventType] = &eventTypeStats{}
	}
	c.typeStats[eventType].processed++
}

func (c *WebSocketConsumer) trackTypeError(eventType string) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	if c.typeStats[eventType] == nil {
		c.typeStats[eventType] = &eventTypeStats{}
	}
	c.typeStats[eventType].errors++
}
