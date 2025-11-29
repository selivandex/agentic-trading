package consumers

import (
	"context"
	"encoding/json"
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
	depthBatchSize     = 50  // Smaller batch for depth data (larger payloads)
	depthFlushInterval = 3 * time.Second
	depthStatsInterval = 1 * time.Minute
)

// WebSocketDepthConsumer consumes depth/orderbook events from Kafka and writes to ClickHouse
type WebSocketDepthConsumer struct {
	consumer   *kafka.Consumer
	repository market_data.Repository
	log        *logger.Logger

	mu    sync.Mutex
	batch []*market_data.OrderBookSnapshot

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

// NewWebSocketDepthConsumer creates a new depth consumer
func NewWebSocketDepthConsumer(
	consumer *kafka.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
) *WebSocketDepthConsumer {
	now := time.Now()
	return &WebSocketDepthConsumer{
		consumer:         consumer,
		repository:       repository,
		log:              log,
		batch:            make([]*market_data.OrderBookSnapshot, 0, depthBatchSize),
		lastFlushTime:    now,
		lastStatsLogTime: now,
	}
}

// Start begins consuming depth events
func (c *WebSocketDepthConsumer) Start(ctx context.Context) error {
	c.log.Info("Starting WebSocket depth consumer...",
		"batch_size", depthBatchSize,
		"flush_interval", depthFlushInterval,
	)

	flushTicker := time.NewTicker(depthFlushInterval)
	defer flushTicker.Stop()

	statsTicker := time.NewTicker(depthStatsInterval)
	defer statsTicker.Stop()

	defer func() {
		c.log.Info("Closing WebSocket depth consumer...")
		if err := c.flushBatch(context.Background()); err != nil {
			c.log.Error("Failed to flush final batch", "error", err)
		}
		c.logStats(true)
		if err := c.consumer.Close(); err != nil {
			c.log.Error("Failed to close consumer", "error", err)
		} else {
			c.log.Info("✓ WebSocket depth consumer closed")
		}
	}()

	go c.periodicFlush(ctx, flushTicker.C)
	go c.periodicStatsLog(ctx, statsTicker.C)

	c.log.Info("🔄 Starting to read messages from Kafka...",
		"topic", "websocket.depth",
	)

	for {
		msg, err := c.consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("Depth consumer stopping (context cancelled)")
				return nil
			}
			c.log.Error("Failed to read depth event", "error", err)
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
		if err := c.handleDepthEvent(processCtx, msg.Value); err != nil {
			c.log.Error("Failed to handle depth event", "error", err)
			c.incrementStat(&c.totalErrors)
		}
		cancel()

		if ctx.Err() != nil {
			c.log.Info("Depth consumer stopping after processing current message")
			return nil
		}
	}
}

func (c *WebSocketDepthConsumer) handleDepthEvent(ctx context.Context, data []byte) error {
	var event eventspb.WebSocketDepthEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		return errors.Wrap(err, "unmarshal depth event")
	}

	snapshot := c.convertProtobufToSnapshot(&event)
	c.addToBatch(snapshot)

	c.log.Debug("Added depth snapshot to batch",
		"exchange", event.Exchange,
		"symbol", event.Symbol,
		"bids_count", len(event.Bids),
		"asks_count", len(event.Asks),
		"batch_size", len(c.batch),
	)

	if len(c.batch) >= depthBatchSize {
		return c.flushBatch(ctx)
	}

	return nil
}

func (c *WebSocketDepthConsumer) addToBatch(snapshot *market_data.OrderBookSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.batch = append(c.batch, snapshot)
}

func (c *WebSocketDepthConsumer) flushBatch(ctx context.Context) error {
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

	if err := c.repository.InsertOrderBook(ctx, dedupedBatch); err != nil {
		c.log.Error("Failed to insert depth batch", "error", err)
		return errors.Wrap(err, "insert batch to ClickHouse")
	}

	duration := time.Since(start)

	c.log.Info("✅ Flushed depth batch to ClickHouse",
		"original_size", originalSize,
		"written_size", len(dedupedBatch),
		"deduplicated", deduplicatedCount,
		"duration_ms", duration.Milliseconds(),
	)

	c.statsMu.Lock()
	c.totalProcessed += int64(len(dedupedBatch))
	c.processedSinceLastLog += int64(len(dedupedBatch))
	c.lastFlushTime = time.Now()
	c.statsMu.Unlock()

	c.batch = c.batch[:0]

	return nil
}

func (c *WebSocketDepthConsumer) deduplicateBatch(batch []*market_data.OrderBookSnapshot) []market_data.OrderBookSnapshot {
	if len(batch) == 0 {
		return []market_data.OrderBookSnapshot{}
	}

	seen := make(map[string]*market_data.OrderBookSnapshot, len(batch))

	for _, snapshot := range batch {
		key := snapshot.Exchange + "_" + snapshot.Symbol + "_" + strconv.FormatInt(snapshot.Timestamp.Unix(), 10)

		existing, exists := seen[key]
		if !exists {
			seen[key] = snapshot
		} else {
			// Keep the snapshot with the latest event time
			if snapshot.EventTime.After(existing.EventTime) {
				seen[key] = snapshot
			}
		}
	}

	result := make([]market_data.OrderBookSnapshot, 0, len(seen))
	for _, snapshot := range seen {
		result = append(result, *snapshot)
	}

	return result
}

func (c *WebSocketDepthConsumer) periodicFlush(ctx context.Context, ticker <-chan time.Time) {
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

func (c *WebSocketDepthConsumer) periodicStatsLog(ctx context.Context, ticker <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:
			c.logStats(false)
		}
	}
}

func (c *WebSocketDepthConsumer) logStats(isFinal bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	prefix := "📊"
	if isFinal {
		prefix = "📊 [FINAL]"
	}

	timeSinceLastLog := time.Since(c.lastStatsLogTime)
	receivedRate := float64(c.receivedSinceLastLog) / timeSinceLastLog.Seconds()
	processedRate := float64(c.processedSinceLastLog) / timeSinceLastLog.Seconds()

	c.log.Info(prefix+" Depth consumer stats",
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

func (c *WebSocketDepthConsumer) incrementStat(counter *int64) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	*counter++
}

func (c *WebSocketDepthConsumer) convertProtobufToSnapshot(event *eventspb.WebSocketDepthEvent) *market_data.OrderBookSnapshot {
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

	// Default to futures (market_type field removed from proto)
	marketType := "futures"

	return &market_data.OrderBookSnapshot{
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

