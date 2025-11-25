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

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		GroupID:  cfg.GroupID,
		Topic:    cfg.Topic,
		MinBytes: cfg.MinBytes,
		MaxBytes: cfg.MaxBytes,
	})

	return &Consumer{
		reader: reader,
		log:    logger.Get().With("component", "kafka_consumer", "topic", cfg.Topic),
	}
}

// MessageHandler is a function that processes a message
type MessageHandler func(ctx context.Context, msg kafka.Message) error

// Consume starts consuming messages and calling the handler
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	c.log.Info("Starting consumer...")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Consumer stopped")
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
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
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
