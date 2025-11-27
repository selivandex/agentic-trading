package consumers

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"

	kafkaadapter "prometheus/internal/adapters/kafka"
	"prometheus/internal/domain/ai_usage"
	eventspb "prometheus/internal/events/proto"
	chrepo "prometheus/internal/repository/clickhouse"
	"prometheus/pkg/errors"
	"prometheus/pkg/logger"
)

// AIUsageConsumer reads AI usage events from Kafka and writes to ClickHouse in batches
// This decouples the main app from ClickHouse and provides reliable delivery
type AIUsageConsumer struct {
	consumer    *kafkaadapter.Consumer
	aiUsageRepo *chrepo.AIUsageRepository
	log         *logger.Logger
}

// NewAIUsageConsumer creates a new AI usage consumer
func NewAIUsageConsumer(
	consumer *kafkaadapter.Consumer,
	aiUsageRepo *chrepo.AIUsageRepository,
	log *logger.Logger,
) *AIUsageConsumer {
	return &AIUsageConsumer{
		consumer:    consumer,
		aiUsageRepo: aiUsageRepo,
		log:         log,
	}
}

// Start begins consuming AI usage events
func (c *AIUsageConsumer) Start(ctx context.Context) error {
	c.log.Info("Starting AI usage consumer (writes to ClickHouse in batches)...")

	// Start batch writer background loop
	c.aiUsageRepo.Start(ctx)

	// Ensure consumer is closed on exit
	defer func() {
		c.log.Info("Closing AI usage consumer...")
		if err := c.consumer.Close(); err != nil {
			c.log.Error("Failed to close AI usage consumer", "error", err)
		} else {
			c.log.Info("✓ AI usage consumer closed")
		}
	}()

	// Ensure batch writer stops gracefully on any exit path
	defer func() {
		c.log.Info("Stopping AI usage batch writer...")
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := c.aiUsageRepo.Stop(stopCtx); err != nil {
			c.log.Error("Failed to stop AI usage batch writer", "error", err)
		} else {
			c.log.Info("✓ AI usage batch writer stopped")
		}
	}()

	// Consume messages from Kafka
	for {
		msg, err := c.consumer.ReadMessage(ctx)
		if err != nil {
			// Check if error is due to context cancellation or reader closure
			if ctx.Err() != nil {
				c.log.Info("AI usage consumer stopping (context cancelled)")
				return nil
			}
			// Reader might be closed during shutdown
			c.log.Debug("Failed to read AI usage event", "error", err)
			continue
		}

		// Process event with timeout to prevent hanging during shutdown
		// Allow up to 5s to complete current message processing
		processCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.handleUsageEvent(processCtx, msg); err != nil {
			c.log.Error("Failed to handle AI usage event",
				"topic", msg.Topic,
				"error", err,
			)
		}
		cancel()

		// Check if we should stop AFTER processing current message
		if ctx.Err() != nil {
			c.log.Info("AI usage consumer stopping after processing current message")
			return nil
		}
	}
}

// handleUsageEvent processes a single AI usage event
func (c *AIUsageConsumer) handleUsageEvent(ctx context.Context, msg kafka.Message) error {
	c.log.Debug("Processing AI usage event",
		"topic", msg.Topic,
		"size", len(msg.Value),
	)

	// Deserialize protobuf event
	var event eventspb.AIUsageEvent
	if err := proto.Unmarshal(msg.Value, &event); err != nil {
		return errors.Wrap(err, "unmarshal ai_usage event")
	}

	// Convert protobuf event to domain entity
	usageLog := &ai_usage.UsageLog{
		Timestamp:        event.Base.Timestamp.AsTime(),
		EventID:          event.Base.Id,
		UserID:           event.Base.UserId,
		SessionID:        event.SessionId,
		AgentName:        event.AgentName,
		AgentType:        event.AgentType,
		Provider:         event.Provider,
		ModelID:          event.ModelId,
		ModelFamily:      event.ModelFamily,
		PromptTokens:     event.PromptTokens,
		CompletionTokens: event.CompletionTokens,
		TotalTokens:      event.TotalTokens,
		InputCostUSD:     event.InputCostUsd,
		OutputCostUSD:    event.OutputCostUsd,
		TotalCostUSD:     event.TotalCostUsd,
		ToolCallsCount:   uint16(event.ToolCallsCount),
		IsCached:         event.IsCached,
		CacheHit:         event.CacheHit,
		LatencyMs:        event.LatencyMs,
		ReasoningStep:    uint16(event.ReasoningStep),
		WorkflowName:     event.WorkflowName,
		CreatedAt:        time.Now(),
	}

	// Add to batch writer (buffered, will flush when batch is full or timeout)
	if err := c.aiUsageRepo.Store(ctx, usageLog); err != nil {
		return errors.Wrap(err, "failed to store AI usage log")
	}

	c.log.Debug("AI usage event buffered for batch insert",
		"agent", event.AgentName,
		"provider", event.Provider,
		"model", event.ModelId,
		"tokens", event.TotalTokens,
		"cost_usd", event.TotalCostUsd,
	)

	return nil
}
