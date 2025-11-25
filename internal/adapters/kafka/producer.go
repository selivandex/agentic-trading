package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"prometheus/pkg/logger"
)

// Producer handles Kafka message publishing
type Producer struct {
	writers map[string]*kafka.Writer
	brokers []string
	log     *logger.Logger
}

// ProducerConfig holds producer configuration
type ProducerConfig struct {
	Brokers []string
	Async   bool
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg ProducerConfig) *Producer {
	return &Producer{
		writers: make(map[string]*kafka.Writer),
		brokers: cfg.Brokers,
		log:     logger.Get().With("component", "kafka_producer"),
	}
}

// getWriter returns or creates a writer for a topic
func (p *Producer) getWriter(topic string) *kafka.Writer {
	if w, ok := p.writers[topic]; ok {
		return w
	}

	w := &kafka.Writer{
		Addr:     kafka.TCP(p.brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
		Async:    false, // Synchronous by default for reliability
	}

	p.writers[topic] = w
	return w
}

// Publish sends a message to a topic
func (p *Producer) Publish(ctx context.Context, topic string, key string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
	}

	if err := p.getWriter(topic).WriteMessages(ctx, msg); err != nil {
		p.log.Errorf("Failed to publish to %s: %v", topic, err)
		return err
	}

	p.log.Debugf("Published to %s: %s", topic, key)
	return nil
}

// PublishBatch sends multiple messages to a topic
func (p *Producer) PublishBatch(ctx context.Context, topic string, messages []kafka.Message) error {
	if err := p.getWriter(topic).WriteMessages(ctx, messages...); err != nil {
		p.log.Errorf("Failed to publish batch to %s: %v", topic, err)
		return err
	}

	p.log.Debugf("Published %d messages to %s", len(messages), topic)
	return nil
}

// Close closes all writers
func (p *Producer) Close() error {
	for topic, w := range p.writers {
		if err := w.Close(); err != nil {
			p.log.Errorf("Failed to close writer for %s: %v", topic, err)
			return err
		}
	}
	return nil
}
