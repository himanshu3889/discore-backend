package baseKafka

// https://medium.com/@harshithgowdakt/kafka-with-confluent-kafka-go-a-go-developers-playbook-30f4993f5248
import (
	"context"
	"discore/internal/base/utils"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Producer sends any struct to any topic
type KafkaProducer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.Hash{}, // Keep this - ensures same key → same partition

			// SPEED OPTIMIZATION 1: Don't wait for all replicas (local dev only!)
			// Use RequireOne for speed, RequireAll for safety
			RequiredAcks: kafka.RequireOne, // Change to RequireAll if you need durability

			// SPEED OPTIMIZATION 2: Batch settings
			BatchSize:    100,                   // Accumulate 100 messages before sending
			BatchTimeout: 10 * time.Millisecond, // Or flush after 10ms

			// SPEED OPTIMIZATION 3: Async mode - Fire and forget (FASTEST)
			// Returns immediately, batches in background
			Async: true,

			// SPEED OPTIMIZATION 4: Compression (essential for batches)
			Compression: kafka.Snappy, // Fast compression, good ratio
			// Or use kafka.Lz4 for higher compression, more CPU

			// Retry failed sends in background
			MaxAttempts: 3,

			// Error handling for async mode (CRITICAL)
			// Async failures go here instead of returning error
			ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
				logrus.Errorf("[KAFKA-PRODUCER] "+msg, args...)
			}),
		},
	}
}

// Send puts any data on any topic.
// Key is used for partitioning (same key = same partition = ordering)
func (p *KafkaProducer) Send(ctx context.Context, topic, key string, data []byte, userID snowflake.ID) error {

	snowflakeID := utils.GenerateSnowflakeID()
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
		Headers: []kafka.Header{
			{Key: "ID", Value: []byte(snowflakeID.String())}, // Unique snowflake id
			{Key: "timestamp", Value: []byte(time.Now().Format(time.RFC3339))},
			{Key: "userID", Value: []byte(userID.String())},
		},
	})
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
