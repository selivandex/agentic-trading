package consumers

import (
	"context"
	"strconv"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	eventspb "prometheus/internal/events/proto"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

const (
	tradeBatchSize     = 500 // Larger batch - trades come very frequently
	tradeFlushInterval = 3 * time.Second
	tradeStatsInterval = 1 * time.Minute
)

// WebSocketTradeConsumer consumes trade events from Kafka and writes to ClickHouse
type WebSocketTradeConsumer struct {
	consumer   *kafka.Consumer
	repository market_data.Repository
	log        *logger.Logger

	mu    sync.Mutex
	batch []*market_data.Trade

	statsMu               sync.Mutex
	totalReceived         int64
	totalProcessed        int64
	totalDeduplicated     int64
	totalErrors           int64
	lastFlushTime         time.Time
	lastStatsLogTime      time.Time
	receivedSinceLastLog  int64
	processedSinceLastLog int64
}

// NewWebSocketTradeConsumer creates a new trade consumer
func NewWebSocketTradeConsumer(
	consumer *kafka.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketTradeConsumer {
	now := time.Now()
	return &WebSocketTradeConsumer{
		consumer:         consumer,
		repository:       repository,
		log:              log,
		batch:            make([]*market_data.Trade, 0, tradeBatchSize),
		lastFlushTime:    now,
		lastStatsLogTime: now,
	}
}

// Start begins consuming trade events
func (c *WebSocketTradeConsumer) Start(ctx context.Context) error {
	c.log.Info("Starting WebSocket trade consumer...",
		"batch_size", tradeBatchSize,
		"flush_interval", tradeFlushInterval,
	)

	flushTicker := time.NewTicker(tradeFlushInterval)
	defer flushTicker.Stop()

	statsTicker := time.NewTicker(tradeStatsInterval)
	defer statsTicker.Stop()

	defer func() {
		c.log.Info("Closing WebSocket trade consumer...")
		if err := c.flushBatch(context.Background()); err != nil {
			c.log.Error("Failed to flush final batch", "error", err)
		}
		c.logStats(true)
		if err := c.consumer.Close(); err != nil {
			c.log.Error("Failed to close consumer", "error", err)
		} else {
			c.log.Info("✓ WebSocket trade consumer closed")
		}
	}()

	go c.periodicFlush(ctx, flushTicker.C)
	go c.periodicStatsLog(ctx, statsTicker.C)

	c.log.Info("🔄 Starting to read messages from Kafka...",
		"topic", "websocket.trade",
	)

	for {
		msg, err := c.consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("Trade consumer stopping (context cancelled)")
				return nil
			}
			c.log.Error("Failed to read trade event", "error", err)
			continue
		}

		c.incrementStat(&c.totalReceived)
		c.incrementStat(&c.receivedSinceLastLog)

		c.log.Debug("📩 Received message from Kafka",
			"partition", msg.Partition,
			"offset", msg.Offset,
			"size_bytes", len(msg.Value),
		)

		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.handleTradeEvent(processCtx, msg.Value); err != nil {
			c.log.Error("Failed to handle trade event", "error", err)
			c.incrementStat(&c.totalErrors)
		}
		cancel()

		if ctx.Err() != nil {
			c.log.Info("Trade consumer stopping after processing current message")
			return nil
		}
	}
}

func (c *WebSocketTradeConsumer) handleTradeEvent(ctx context.Context, data []byte) error {
	var event eventspb.WebSocketTradeEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal trade event")
	}

	trade := c.convertProtobufToTrade(&event)
	c.addToBatch(trade)

	c.log.Debug("Added trade to batch",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
		"price", event.Price,
		"quantity", event.Quantity,
		"batch_size", len(c.batch),
	)

	if len(c.batch) >= tradeBatchSize {
		return c.flushBatch(ctx)
	}

	return nil
}

func (c *WebSocketTradeConsumer) addToBatch(trade *market_data.Trade) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.batch = append(c.batch, trade)
}

func (c *WebSocketTradeConsumer) flushBatch(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.batch) == 0 {
		return nil
	}

	originalSize := len(c.batch)
	start := time.Now()

	dedupedBatch := c.deduplicateBatch(c.batch)
	deduplicatedCount := originalSize - len(dedupedBatch)

	if deduplicatedCount > 0 {
		c.statsMu.Lock()
		c.totalDeduplicated += int64(deduplicatedCount)
		c.statsMu.Unlock()

		c.log.Debug("Deduplicated batch before write",
			"original_size", originalSize,
			"deduplicated_size", len(dedupedBatch),
			"removed", deduplicatedCount,
		)
	}

	if err := c.repository.InsertTrades(ctx, dedupedBatch); err != nil {
		c.log.Error("Failed to insert trade batch", "error", err)
		return errors.Wrap(err, "insert batch to ClickHouse")
	}

	duration := time.Since(start)

	c.log.Info("✅ Flushed trade batch to ClickHouse",
		"original_size", originalSize,
		"written_size", len(dedupedBatch),
		"deduplicated", deduplicatedCount,
		"duration_ms", duration.Milliseconds(),
		"trades_per_sec", int64(float64(len(dedupedBatch))/duration.Seconds()),
	)

	c.statsMu.Lock()
	c.totalProcessed += int64(len(dedupedBatch))
	c.processedSinceLastLog += int64(len(dedupedBatch))
	c.lastFlushTime = time.Now()
	c.statsMu.Unlock()

	c.batch = c.batch[:0]

	return nil
}

func (c *WebSocketTradeConsumer) deduplicateBatch(batch []*market_data.Trade) []market_data.Trade {
	if len(batch) == 0 {
		return []market_data.Trade{}
	}

	seen := make(map[string]*market_data.Trade, len(batch))

	for _, trade := range batch {
		key := trade.Exchange + "_" + trade.Symbol + "_" + strconv.FormatInt(trade.TradeID, 10)

		existing, exists := seen[key]
		if !exists {
			seen[key] = trade
		} else {
			if trade.EventTime.After(existing.EventTime) {
				seen[key] = trade
			}
		}
	}

	result := make([]market_data.Trade, 0, len(seen))
	for _, trade := range seen {
		result = append(result, *trade)
	}

	return result
}

func (c *WebSocketTradeConsumer) periodicFlush(ctx context.Context, ticker <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			if err := c.flushBatch(ctx); err != nil {
				c.log.Error("Periodic flush failed", "error", err)
			}
		}
	}
}

func (c *WebSocketTradeConsumer) periodicStatsLog(ctx context.Context, ticker <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			c.logStats(false)
		}
	}
}

func (c *WebSocketTradeConsumer) logStats(isFinal bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	prefix := "📊"
	if isFinal {
		prefix = "📊 [FINAL]"
	}

	timeSinceLastLog := time.Since(c.lastStatsLogTime)
	receivedRate := float64(c.receivedSinceLastLog) / timeSinceLastLog.Seconds()
	processedRate := float64(c.processedSinceLastLog) / timeSinceLastLog.Seconds()

	c.log.Info(prefix+" Trade consumer stats",
		"total_received", c.totalReceived,
		"total_processed", c.totalProcessed,
		"total_deduplicated", c.totalDeduplicated,
		"total_errors", c.totalErrors,
		"pending_batch", len(c.batch),
		"received_per_sec", int64(receivedRate),
		"processed_per_sec", int64(processedRate),
		"time_since_last_flush", time.Since(c.lastFlushTime).Round(time.Second),
	)

	c.receivedSinceLastLog = 0
	c.processedSinceLastLog = 0
	c.lastStatsLogTime = time.Now()
}

func (c *WebSocketTradeConsumer) incrementStat(counter *int64) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	*counter++
}

func (c *WebSocketTradeConsumer) convertProtobufToTrade(event *eventspb.WebSocketTradeEvent) *market_data.Trade {
	parseFloat := func(s string) float64 {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}

	// Use market_type from event, fallback to "futures" for backward compatibility
	marketType := event.MarketType
	if marketType == "" {
		marketType = "futures"
	}

	return &market_data.Trade{
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
