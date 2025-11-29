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
	liquidationBatchSize     = 50 // Smaller batch - liquidations are less frequent but important
	liquidationFlushInterval = 5 * time.Second
	liquidationStatsInterval = 1 * time.Minute
)

// WebSocketLiquidationConsumer consumes liquidation events from Kafka and writes to ClickHouse
type WebSocketLiquidationConsumer struct {
	consumer   *kafka.Consumer
	repository market_data.Repository
	log        *logger.Logger

	mu    sync.Mutex
	batch []*market_data.Liquidation

	statsMu               sync.Mutex
	totalReceived         int64
	totalProcessed        int64
	totalErrors           int64
	lastFlushTime         time.Time
	lastStatsLogTime      time.Time
	receivedSinceLastLog  int64
	processedSinceLastLog int64
}

// NewWebSocketLiquidationConsumer creates a new liquidation consumer
func NewWebSocketLiquidationConsumer(
	consumer *kafka.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketLiquidationConsumer {
	now := time.Now()
	return &WebSocketLiquidationConsumer{
		consumer:         consumer,
		repository:       repository,
		log:              log,
		batch:            make([]*market_data.Liquidation, 0, liquidationBatchSize),
		lastFlushTime:    now,
		lastStatsLogTime: now,
	}
}

// Start begins consuming liquidation events
func (c *WebSocketLiquidationConsumer) Start(ctx context.Context) error {
	c.log.Info("Starting WebSocket liquidation consumer...",
		"batch_size", liquidationBatchSize,
		"flush_interval", liquidationFlushInterval,
	)

	flushTicker := time.NewTicker(liquidationFlushInterval)
	defer flushTicker.Stop()

	statsTicker := time.NewTicker(liquidationStatsInterval)
	defer statsTicker.Stop()

	defer func() {
		c.log.Info("Closing WebSocket liquidation consumer...")
		if err := c.flushBatch(context.Background()); err != nil {
			c.log.Error("Failed to flush final batch", "error", err)
		}
		c.logStats(true)
		if err := c.consumer.Close(); err != nil {
			c.log.Error("Failed to close consumer", "error", err)
		} else {
			c.log.Info("✓ WebSocket liquidation consumer closed")
		}
	}()

	go c.periodicFlush(ctx, flushTicker.C)
	go c.periodicStatsLog(ctx, statsTicker.C)

	c.log.Info("🔄 Starting to read messages from Kafka...",
		"topic", "websocket.liquidation",
	)

	for {
		msg, err := c.consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("Liquidation consumer stopping (context cancelled)")
				return nil
			}
			c.log.Error("Failed to read liquidation event", "error", err)
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
		if err := c.handleLiquidationEvent(processCtx, msg.Value); err != nil {
			c.log.Error("Failed to handle liquidation event", "error", err)
			c.incrementStat(&c.totalErrors)
		}
		cancel()

		if ctx.Err() != nil {
			c.log.Info("Liquidation consumer stopping after processing current message")
			return nil
		}
	}
}

func (c *WebSocketLiquidationConsumer) handleLiquidationEvent(ctx context.Context, data []byte) error {
	var event eventspb.WebSocketLiquidationEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal liquidation event")
	}

	liquidation := c.convertProtobufToLiquidation(&event)
	c.addToBatch(liquidation)

	c.log.Debug("Added liquidation to batch",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
		"side", event.Side,
		"value", event.Value,
		"batch_size", len(c.batch),
	)

	if len(c.batch) >= liquidationBatchSize {
		return c.flushBatch(ctx)
	}

	return nil
}

func (c *WebSocketLiquidationConsumer) convertProtobufToLiquidation(event *eventspb.WebSocketLiquidationEvent) *market_data.Liquidation {
	price, _ := strconv.ParseFloat(event.Price, 64)
	quantity, _ := strconv.ParseFloat(event.Quantity, 64)
	value, _ := strconv.ParseFloat(event.Value, 64)

	return &market_data.Liquidation{
		Exchange:   event.Exchange,
		Symbol:     event.Symbol,
		MarketType: event.MarketType,
		Timestamp:  event.EventTime.AsTime(),
		Side:       event.Side,
		OrderType:  event.OrderType,
		Price:      price,
		Quantity:   quantity,
		Value:      value,
		EventTime:  event.EventTime.AsTime(),
	}
}

func (c *WebSocketLiquidationConsumer) addToBatch(liquidation *market_data.Liquidation) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.batch = append(c.batch, liquidation)
}

func (c *WebSocketLiquidationConsumer) flushBatch(ctx context.Context) error {
	c.mu.Lock()
	if len(c.batch) == 0 {
		c.mu.Unlock()
		return nil
	}

	batch := c.batch
	c.batch = make([]*market_data.Liquidation, 0, liquidationBatchSize)
	c.mu.Unlock()

	c.log.Debug("💾 Flushing liquidation batch to ClickHouse",
		"batch_size", len(batch),
	)

	start := time.Now()

	// Convert pointers to values for repository
	liquidations := make([]market_data.Liquidation, len(batch))
	for i, liq := range batch {
		liquidations[i] = *liq
	}

	if err := c.repository.InsertLiquidations(ctx, liquidations); err != nil {
		c.log.Error("Failed to insert liquidation batch", "error", err)
		return errors.Wrap(err, "insert batch to ClickHouse")
	}

	duration := time.Since(start)
	c.incrementStat(&c.totalProcessed)

	c.statsMu.Lock()
	c.lastFlushTime = time.Now()
	c.processedSinceLastLog += int64(len(batch))
	c.statsMu.Unlock()

	c.log.Debug("✅ Liquidation batch flushed successfully",
		"batch_size", len(batch),
		"duration_ms", duration.Milliseconds(),
	)

	return nil
}

func (c *WebSocketLiquidationConsumer) periodicFlush(ctx context.Context, ticker <-chan time.Time) {
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

func (c *WebSocketLiquidationConsumer) periodicStatsLog(ctx context.Context, ticker <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			c.logStats(false)
		}
	}
}

func (c *WebSocketLiquidationConsumer) logStats(final bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	now := time.Now()
	timeSinceLastLog := now.Sub(c.lastStatsLogTime)

	receivedRate := float64(c.receivedSinceLastLog) / timeSinceLastLog.Seconds()
	processedRate := float64(c.processedSinceLastLog) / timeSinceLastLog.Seconds()

	prefix := "📊"
	if final {
		prefix = "🏁"
	}

	c.log.Info(prefix+" WebSocket liquidation consumer stats",
		"total_received", c.totalReceived,
		"total_processed", c.totalProcessed,
		"total_errors", c.totalErrors,
		"received_rate_per_sec", receivedRate,
		"processed_rate_per_sec", processedRate,
		"current_batch_size", len(c.batch),
	)

	c.receivedSinceLastLog = 0
	c.processedSinceLastLog = 0
	c.lastStatsLogTime = now
}

func (c *WebSocketLiquidationConsumer) incrementStat(counter *int64) {
	c.statsMu.Lock()
	*counter++
	c.statsMu.Unlock()
}

