package ChatkafkaService

import (
	"context"
	"encoding/json"
	"time"

	baseKafka "github.com/himanshu3889/discore-backend/base/infrastructure/kafka"
	"github.com/himanshu3889/discore-backend/base/models"
	channelMessageStore "github.com/himanshu3889/discore-backend/base/store/channelMessage"

	"github.com/bwmarrin/snowflake"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Closure to handle the kafka consumer message
func MakeChannelMessageHandler(producer *baseKafka.KafkaProducer) func(*kafka.Message) error {
	return func(msg *kafka.Message) error {
		// logrus.Infof("RAW JSON in channel handler consumer: %s\n", string(msg.Value))
		metadata := baseKafka.ParseKafkaMessageHeaders(msg)

		_, err := HandleChannelByteMessage(msg.Value, metadata.TraceID, metadata.UserID, metadata.IngestTime)
		return err
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
