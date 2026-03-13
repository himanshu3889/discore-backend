package ChatkafkaService

import (
	"context"
	"encoding/json"
	"errors"
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

// MakeChannelMessageHandler creates a closure to handle a batch of Kafka messages
func MakeChannelMessagesHandler(producer *baseKafka.KafkaProducer) func([]*kafka.Message) (error, []*kafka.Message) {
	return func(messages []*kafka.Message) (error, []*kafka.Message) {
		var modelsToInsert []*models.ChannelMessage
		var dlq []*kafka.Message
		var validMessages []*kafka.Message

		for _, msg := range messages {
			metadata := baseKafka.ParseKafkaMessageHeaders(msg)
			parsedMsg, err := ParseChannelByteMessage(msg.Value, metadata.TraceID, metadata.UserID, metadata.IngestTime)
			if err != nil {
				dlq = append(dlq, msg)
				continue
			}
			modelsToInsert = append(modelsToInsert, parsedMsg)
			validMessages = append(validMessages, msg)
		}

		if len(modelsToInsert) == 0 {
			return nil, dlq
		}

		// Bulk insert into the database
		ctx := context.Background()
		failedMsgIndices, appErr := channelMessageStore.CreateChannelMessagesBulk(ctx, modelsToInsert)

		// Manage the dlq
		for _, idx := range failedMsgIndices {
			dlq = append(dlq, validMessages[idx])
		}

		if appErr != nil {
			return errors.New(appErr.Message), dlq
		}

		return nil, dlq
	}
}

// unmarshals and formats the message WITHOUT saving to the DB
func ParseChannelByteMessage(msg []byte, ID snowflake.ID, userID snowflake.ID, createdAt time.Time) (*models.ChannelMessage, error) {
	var incomingMessage models.ChannelMessage

	if err := json.Unmarshal(msg, &incomingMessage); err != nil {
		logrus.WithError(err).Warn("Invalid message format")
		return nil, err
	}

	incomingMessage.ID = ID
	incomingMessage.UserID = userID
	incomingMessage.CreatedAt = createdAt

	return &incomingMessage, nil
}

// Handle the raw message in the channel
func HandleChannelByteMessage(msg []byte, ID snowflake.ID, userID snowflake.ID, createdAt time.Time) (*models.ChannelMessage, error) {
	incomingMessage, err := ParseChannelByteMessage(msg, ID, userID, createdAt)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	message, appErr := channelMessageStore.CreateChannelMessage(ctx, incomingMessage)
	if appErr != nil {
		return nil, errors.New(appErr.Message)
	}
	return message, nil

}
