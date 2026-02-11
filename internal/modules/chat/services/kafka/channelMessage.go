package ChatkafkaService

import (
	"context"
	baseKafka "discore/internal/base/infrastructure/kafka"
	"discore/internal/modules/chat/models"
	channelMessageStore "discore/internal/modules/chat/store/channelMessage"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Closure to handle the kafka consumer message
func MakeChannelMessageHandler(producer *baseKafka.KafkaProducer) func(*kafka.Message) error {
	return func(msg *kafka.Message) error {
		// logrus.Infof("RAW JSON in channel handler consumer: %s\n", string(msg.Value))
		metadata := baseKafka.ParseKafkaMessageHeaders(msg)
		// logrus.Info(metadata)

		createdMessage, err := HandleChannelByteMessage(msg.Value, metadata.ID, metadata.UserID, metadata.Timestamp)
		if err != nil {
			return err
		}

		createdMessageBytes, err := json.Marshal(createdMessage)
		if err != nil {
			return fmt.Errorf("marshal failed: %w", err)
		}

		// Produce to NEXT topic (e.g., "broadcast" or "notifications")
		broadcastTopic := "broadcast." + msg.Topic
		err = producer.Send(context.Background(), broadcastTopic,
			string(msg.Key),
			createdMessageBytes,
			metadata.UserID,
		)

		if err != nil {
			// logrus.WithError(err).Error("Failed to forward to broadcast topic")
			return err // Return error so Kafka retries
		}

		return nil
	}
}

// Handle the raw message in the channel
func HandleChannelByteMessage(msg []byte, ID snowflake.ID, userID snowflake.ID, createdAt time.Time) (*models.ChannelMessage, error) {
	var incomingMessage models.ChannelMessage

	if err := json.Unmarshal(msg, &incomingMessage); err != nil {
		logrus.WithError(err).Warn("Invalid message format")
		return nil, err
	}
	incomingMessage.ID = ID
	incomingMessage.UserID = userID
	incomingMessage.CreatedAt = createdAt

	ctx := context.Background()
	message, err := channelMessageStore.CreateChannelMessage(ctx, &incomingMessage)
	if err != nil {
		return nil, err
	}
	return message, nil

}
