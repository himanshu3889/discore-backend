package channelMessage

import (
	"context"
	"discore/internal/modules/chat/models"
	"discore/internal/modules/chat/store/message"
	"encoding/json"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Handle the raw message in the channel
func HandleChannelRawMessage(userID snowflake.ID, msg json.RawMessage) (*models.ChannelMessage, error) {
	var incomingMessage models.ChannelMessage

	if err := json.Unmarshal(msg, &incomingMessage); err != nil {
		logrus.WithError(err).Warn("Invalid message format")
		return nil, err
	}
	incomingMessage.UserID = userID

	ctx := context.Background()
	message, err := message.CreateChannelMessage(ctx, &incomingMessage)
	if err != nil {
		logrus.WithError(err).Error("Unable to create message")
		return nil, err
	}
	logrus.WithFields(logrus.Fields{"message": message}).Info("Message Created")
	return message, nil

}
