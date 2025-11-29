package consumers

import (
	"context"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/market_data"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// BatchProcessingConfig holds configuration for batch processing
type BatchProcessingConfig struct {
	ConsumerName  string
	BatchSize     int
	FlushInterval time.Duration
	StatsInterval time.Duration
}

// EventHandler defines the interface for handling specific event types
// T is the protobuf event type (e.g., *eventspb.WebSocketTradeEvent)
// D is the domain model type (e.g., *market_data.Trade)
type EventHandler[T proto.Message, D any] interface {
	// UnmarshalEvent unmarshals protobuf bytes into event type
	UnmarshalEvent(data []byte) (T, error)

	// ConvertToDomain converts protobuf event to domain model
	ConvertToDomain(event T) D

	// GetDeduplicationKey returns unique key for deduplication
	// If empty string is returned, deduplication is disabled
	GetDeduplicationKey(item D) string

	// GetEventTime returns event time for deduplication comparison
	GetEventTime(item D) time.Time

	// InsertBatch writes batch to storage
	InsertBatch(ctx context.Context, batch []D) error

	// GetCustomStats returns custom stats for logging (optional)
	// Return empty map if no custom stats needed
	GetCustomStats() map[string]interface{}
}

// WebSocketBaseConsumer is a generic base consumer for websocket events
// T is the protobuf event type, D is the domain model type
type WebSocketBaseConsumer[T proto.Message, D any] struct {
	consumer   *kafkaadapter.Consumer
	repository market_data.Repository
	log        *logger.Logger
	config     BatchProcessingConfig
	handler    EventHandler[T, D]

	// Batching
	mu    sync.Mutex
	batch []D

	// Statistics
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

// NewWebSocketBaseConsumer creates a new generic websocket consumer
func NewWebSocketBaseConsumer[T proto.Message, D any](
	consumer *kafkaadapter.Consumer,
	repository market_data.Repository,
	log *logger.Logger,
	config BatchProcessingConfig,
	handler EventHandler[T, D],
) *WebSocketBaseConsumer[T, D] {
	now := time.Now()
	return &WebSocketBaseConsumer[T, D]{
		consumer:         consumer,
		repository:       repository,
		log:              log,
		config:           config,
		handler:          handler,
		batch:            make([]D, 0, config.BatchSize),
		lastFlushTime:    now,
		lastStatsLogTime: now,
	}
}

// Start begins consuming events
func (c *WebSocketBaseConsumer[T, D]) Start(ctx context.Context) error {
	c.log.Info("Starting websocket consumer...",
		"consumer", c.config.ConsumerName,
		"batch_size", c.config.BatchSize,
		"flush_interval", c.config.FlushInterval,
	)

	// Use BatchConsumerLifecycle for DRY shutdown/flush management
	lifecycle := NewBatchConsumerLifecycle(
		BatchConsumerConfig{
			ConsumerName:  c.config.ConsumerName,
			FlushInterval: c.config.FlushInterval,
			StatsInterval: c.config.StatsInterval,
			Logger:        c.log,
		},
		c.consumer,
		c, // implements BatchConsumer interface
	)

	// Setup cleanup (flush, stats, close consumer)
	defer lifecycle.Start(ctx)()

	// Start background workers (periodic flush and stats)
	lifecycle.StartBackgroundWorkers(ctx)

	c.log.Info("🔄 Starting to read messages from Kafka...",
		"consumer", c.config.ConsumerName,
	)

	// Use Consume() for DRY message loop with built-in shutdown handling
	return c.consumer.Consume(ctx, c.handleMessage)
}

// handleMessage processes a single Kafka message
func (c *WebSocketBaseConsumer[T, D]) handleMessage(ctx context.Context, msg kafka.Message) error {
	c.incrementStat(&c.totalReceived)
	c.incrementStat(&c.receivedSinceLastLog)

	c.log.Debug("📩 Received message from Kafka",
		"consumer", c.config.ConsumerName,
		"partition", msg.Partition,
		"offset", msg.Offset,
		"size_bytes", len(msg.Value),
	)

	processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.handleEvent(processCtx, msg.Value); err != nil {
		c.log.Error("Failed to handle event",
			"consumer", c.config.ConsumerName,
			"error", err,
		)
		c.incrementStat(&c.totalErrors)
		return err
	}

	return nil
}

// handleEvent processes a single event
func (c *WebSocketBaseConsumer[T, D]) handleEvent(ctx context.Context, data []byte) error {
	// Unmarshal protobuf event
	event, err := c.handler.UnmarshalEvent(data)
	if err != nil {
		return errors.Wrap(err, "unmarshal event")
	}

	// Convert to domain model
	domainItem := c.handler.ConvertToDomain(event)

	// Add to batch
	c.addToBatch(domainItem)

	c.log.Debug("Added event to batch",
		"consumer", c.config.ConsumerName,
		"batch_size", len(c.batch),
	)

	// Flush if batch is full
	if len(c.batch) >= c.config.BatchSize {
		return c.FlushBatch(ctx)
	}

	return nil
}

// addToBatch adds an item to the batch (thread-safe)
func (c *WebSocketBaseConsumer[T, D]) addToBatch(item D) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.batch = append(c.batch, item)
}

// FlushBatch writes the batch to storage (implements BatchConsumer interface)
func (c *WebSocketBaseConsumer[T, D]) FlushBatch(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.batch) == 0 {
		return nil
	}

	originalSize := len(c.batch)
	start := time.Now()

	// Deduplicate batch if handler provides deduplication logic
	dedupedBatch := c.deduplicateBatch(c.batch)
	deduplicatedCount := originalSize - len(dedupedBatch)

	if deduplicatedCount > 0 {
		c.statsMu.Lock()
		c.totalDeduplicated += int64(deduplicatedCount)
		c.statsMu.Unlock()

		c.log.Debug("Deduplicated batch before write",
			"consumer", c.config.ConsumerName,
			"original_size", originalSize,
			"deduplicated_size", len(dedupedBatch),
			"removed", deduplicatedCount,
		)
	}

	// Write to storage
	if err := c.handler.InsertBatch(ctx, dedupedBatch); err != nil {
		c.log.Error("Failed to insert batch",
			"consumer", c.config.ConsumerName,
			"error", err,
		)
		return errors.Wrap(err, "insert batch to storage")
	}

	duration := time.Since(start)

	c.log.Info("✅ Flushed batch successfully",
		"consumer", c.config.ConsumerName,
		"original_size", originalSize,
		"written_size", len(dedupedBatch),
		"deduplicated", deduplicatedCount,
		"duration_ms", duration.Milliseconds(),
	)

	// Update stats
	c.statsMu.Lock()
	c.totalProcessed += int64(len(dedupedBatch))
	c.processedSinceLastLog += int64(len(dedupedBatch))
	c.lastFlushTime = time.Now()
	c.statsMu.Unlock()

	// Clear batch
	c.batch = c.batch[:0]

	return nil
}

// deduplicateBatch keeps only the latest event for each unique key
func (c *WebSocketBaseConsumer[T, D]) deduplicateBatch(batch []D) []D {
	if len(batch) == 0 {
		return batch
	}

	// Check if handler provides deduplication logic
	if len(batch) > 0 {
		// Test if handler provides deduplication key
		testKey := c.handler.GetDeduplicationKey(batch[0])
		if testKey == "" {
			// No deduplication needed
			return batch
		}
	}

	// Map: key -> latest item
	seen := make(map[string]D, len(batch))

	for _, item := range batch {
		key := c.handler.GetDeduplicationKey(item)
		if key == "" {
			// Skip deduplication for items without key
			continue
		}

		existing, exists := seen[key]
		if !exists {
			seen[key] = item
		} else {
			// Keep the one with latest event_time
			if c.handler.GetEventTime(item).After(c.handler.GetEventTime(existing)) {
				seen[key] = item
			}
		}
	}

	// Convert map back to slice
	result := make([]D, 0, len(seen))
	for _, item := range seen {
		result = append(result, item)
	}

	return result
}

// LogStats logs consumer statistics (implements BatchConsumer interface)
func (c *WebSocketBaseConsumer[T, D]) LogStats(isFinal bool) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	prefix := "📊"
	if isFinal {
		prefix = "📊 [FINAL]"
	}

	timeSinceLastLog := time.Since(c.lastStatsLogTime)
	receivedRate := float64(c.receivedSinceLastLog) / timeSinceLastLog.Seconds()
	processedRate := float64(c.processedSinceLastLog) / timeSinceLastLog.Seconds()

	// Log message
	message := prefix + " " + c.config.ConsumerName + " stats"

	// Base log fields as variadic args
	c.log.Info(message,
		"total_received", c.totalReceived,
		"total_processed", c.totalProcessed,
		"total_deduplicated", c.totalDeduplicated,
		"total_errors", c.totalErrors,
		"pending_batch", len(c.batch),
		"received_per_sec", int64(receivedRate),
		"processed_per_sec", int64(processedRate),
		"time_since_last_flush", time.Since(c.lastFlushTime).Round(time.Second),
	)

	// TODO: Custom stats from handler can be logged separately if needed

	// Reset interval counters
	c.receivedSinceLastLog = 0
	c.processedSinceLastLog = 0
	c.lastStatsLogTime = time.Now()
}

// incrementStat safely increments a stat counter
func (c *WebSocketBaseConsumer[T, D]) incrementStat(counter *int64) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()
	*counter++
}
