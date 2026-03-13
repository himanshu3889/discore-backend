package ChatkafkaService

import (
	"context"
	"fmt"

	baseKafka "github.com/himanshu3889/discore-backend/base/infrastructure/kafka"
	"github.com/segmentio/kafka-go"
)

// MakeChannelMessageHandler creates a closure to handle a batch of Kafka messages
func MakeChannelMessagesDLQ(ctx context.Context, producer *baseKafka.KafkaProducer) func([]*kafka.Message) error {
	return func(messages []*kafka.Message) error {
		if len(messages) == 0 {
			return nil
		}

		// Create a slice of values for the Writer
		dlqMessages := make([]kafka.Message, len(messages))

		for i, originalMsg := range messages {
			dlqMessages[i] = kafka.Message{
				Topic:   fmt.Sprintf("dlq.%s", originalMsg.Topic),
				Key:     originalMsg.Key,
				Value:   originalMsg.Value,
				Headers: originalMsg.Headers,
			}
		}

		// Bulk publish all DLQ messages in a single network request
		messagesTopic := messages[0].Topic
		if err := producer.WriteMessages(ctx, messagesTopic, dlqMessages); err != nil {
			return err
		}

		return nil

	}
}
