package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"

	"prometheus/pkg/logger"
)

// Consumer handles Kafka message consumption
type Consumer struct {
	reader *kafka.Reader
	log    *logger.Logger
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Brokers  []string
	GroupID  string
	Topic    string
	MinBytes int
	MaxBytes int
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg ConsumerConfig) *Consumer {
	if cfg.MinBytes == 0 {
		cfg.MinBytes = 10e3 // 10KB
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 10e6 // 10MB
	}

	log := logger.Get().With("component", "kafka_consumer", "topic", cfg.Topic)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		GroupID:     cfg.GroupID,
		Topic:       cfg.Topic,
		MinBytes:    cfg.MinBytes,
		MaxBytes:    cfg.MaxBytes,
		StartOffset: kafka.FirstOffset, // Start from beginning if no offset committed
	})

	log.Info("Kafka consumer created",
		"brokers", cfg.Brokers,
		"group_id", cfg.GroupID,
		"topic", cfg.Topic,
	)

	return &Consumer{
		reader: reader,
		log:    log,
	}
}

// MessageHandler is a function that processes a message
type MessageHandler func(ctx context.Context, msg kafka.Message) error

// Consume starts consuming messages and calling the handler
// Uses ReadMessageWithShutdownCheck internally for graceful shutdown
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	c.log.Info("Starting consumer...")

	for {
		msg, err := c.ReadMessageWithShutdownCheck(ctx)
		if err != nil {
			// Check if shutdown was requested
			if ctx.Err() != nil {
				c.log.Info("Consumer stopped")
				return ctx.Err()
			}
			c.log.Errorf("Failed to read message: %v", err)
			continue
		}

		c.log.Debugf("Received message: key=%s", string(msg.Key))

		if err := handler(ctx, msg); err != nil {
			c.log.Errorf("Failed to handle message: %v", err)
			// Continue processing other messages even if one fails
		}
	}
}

// ReadMessage reads the next message (blocking until message available or ctx cancelled)
func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.ReadMessage(ctx)
}

// ReadMessageWithShutdownCheck reads the next message with explicit shutdown check before blocking.
// This prevents the consumer from blocking on ReadMessage when shutdown is requested.
//
// Returns:
//   - kafka.Message: the read message
//   - error: context.Canceled if shutdown requested, or any read error
//
// Usage pattern:
//
//	for {
//	    msg, err := consumer.ReadMessageWithShutdownCheck(ctx)
//	    if err != nil {
//	        if errors.Is(err, context.Canceled) {
//	            log.Info("Consumer stopping (shutdown requested)")
//	            return nil
//	        }
//	        log.Error("Failed to read message", "error", err)
//	        continue
//	    }
//	    // Process message...
//	}
func (c *Consumer) ReadMessageWithShutdownCheck(ctx context.Context) (kafka.Message, error) {
	// Check for shutdown signal BEFORE blocking on ReadMessage
	// This ensures we don't block on I/O when shutdown is requested
	select {
	case <-ctx.Done():
		return kafka.Message{}, ctx.Err()
	default:
	}

	// Now safe to block on ReadMessage
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		// Double-check if error is due to context cancellation
		if ctx.Err() != nil {
			return kafka.Message{}, ctx.Err()
		}
		return kafka.Message{}, err
	}

	return msg, nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
