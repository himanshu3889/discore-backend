package baseKafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Consumer routes by topic
type Consumer struct {
	reader  *kafka.Reader
	handler func(*kafka.Message) error
}

// TODO: implement the error logger here and metric for that also

// New kafka consumer
func NewConsumer(brokers []string, groupID string, topic string, handler func(*kafka.Message) error) *Consumer {

	// Create a custom Dialer with longer timeouts for Docker/Windows
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: false, //controls whether your app tries both IPv4 and IPv6 to connect to Kafka.
	}

	// Configure the Reader
	config := kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: groupID,
		Topic:   topic,
		// Partition: 0, //

		// Small batches for instant feedback while learning
		MinBytes: 1,
		MaxBytes: 10e6,                   // 10MB max per fetch
		MaxWait:  500 * time.Millisecond, // Faster response than default 1s

		// Group coordination - relaxed for local Docker
		SessionTimeout:    30 * time.Second,
		RebalanceTimeout:  30 * time.Second,
		HeartbeatInterval: 3 * time.Second,

		StartOffset:    kafka.FirstOffset,
		CommitInterval: 1 * time.Second,

		// ATTACH THE DIALER
		Dialer: dialer,

		// ENABLE INTERNAL DEBUGGING
		// This prints directly to stdout so you can see the handshake
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			// logrus.Infof("[KAFKA-DEBUG] "+msg, args...)
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			// logrus.Warnf("[KAFKA-ERROR] "+msg, args...)
		}),
	}

	return &Consumer{
		reader:  kafka.NewReader(config),
		handler: handler,
	}
}

// Start the kafka consumer
func (c *Consumer) Start(ctx context.Context) error {
	logrus.Info("Consumer started, joining group...") // Add this
	for {
		m, err := c.reader.ReadMessage(ctx)

		if err != nil {
			if ctx.Err() != nil {
				return nil // Graceful shutdown
			}
			logrus.Warnf("Read error: %v", err)
			continue
		}
		// logrus.Infof("READ: offset=%d, partition=%d", m.Offset, m.Partition)

		// Process HERE - one by one, in order
		// This blocks if channel full, preventing next fetch
		if err := c.handler(&m); err != nil {
			logrus.Errorf("Handler error: %v", err)
			// Don't commit - will retry
			// In production, send to DLQ here instead of logging
			continue
		}

		// Only commit AFTER handler succeeds (and channel has space)
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			logrus.Errorf("Commit failed: %v", err)
		}
	}
}

// Close the kafka consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
