package message

import (
	"context"
	"discore/internal/base/utils"
	"discore/internal/modules/chat/database"
	"discore/internal/modules/chat/models"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
)

// Create message in the database
func CreateChannelMessage(ctx context.Context, msg *models.ChannelMessage) (*models.ChannelMessage, error) {
	// Set server-side fields
	now := time.Now().UTC()
	msg.ID = utils.GenerateSnowflakeID()
	msg.CreatedAt = &now
	deleted := false
	msg.Deleted = &deleted

	// Validate required fields
	if msg.Content == "" && msg.FileURL == nil {
		logrus.Error("message must have content or file to create message")
		return nil, errors.New("message must have content or file")
	}
	if msg.ServerID == 0 {
		logrus.Error("Server ID is required to create message")
		return nil, errors.New("server ID is required")
	}
	if msg.ChannelID == 0 {
		logrus.Error("channel ID is required to create message")
		return nil, errors.New("channel ID is required")
	}
	if msg.UserID == 0 {
		logrus.Error("user ID is required to create message")
		return nil, errors.New("user ID is required")
	}

	// Insert into database
	_, err := database.MongoDB.Collection("channel_messages").InsertOne(ctx, msg)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"content_length": len(msg.Content),
			"file_url":       msg.FileURL,
			"channel_id":     msg.ChannelID,
			"user_id":        msg.UserID,
		}).WithError(err).Error("Failed to insert message")
		return nil, err
	}

	return msg, nil
}
