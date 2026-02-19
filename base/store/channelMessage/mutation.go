package channelMessageStore

import (
	"context"
	"errors"

	"github.com/himanshu3889/discore-backend/base/databases"
	"github.com/himanshu3889/discore-backend/base/models"

	"github.com/sirupsen/logrus"
)

// Create message in the database
func CreateChannelMessage(ctx context.Context, msg *models.ChannelMessage) (*models.ChannelMessage, error) {
	// Set server-side fields
	deleted := false
	msg.Deleted = &deleted

	// Validate required fields
	// if msg.CreatedAt == nil {
	// 	logrus.Error("Created at is required to create message")
	// 	return nil, errors.New("Created at is required")
	// }
	if msg.ID == 0 {
		logrus.Error("Message ID is required to create message")
		return nil, errors.New("message ID is required")
	}
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
		}).WithError(err).Error("Failed to insert message in channel messages")
		return nil, errors.New("Failed to insert the message in channel messages")
	}

	return msg, nil
}
