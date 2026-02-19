package directMessageStore

import (
	"context"
	"errors"

	"github.com/himanshu3889/discore-backend/base/databases"
	"github.com/himanshu3889/discore-backend/base/models"

	"github.com/sirupsen/logrus"
)

// Create message in the database
func CreateDirectMessage(ctx context.Context, msg *models.DirectMessage) error {
	// Set server-side fields
	deleted := false
	msg.Deleted = &deleted

	// Validate required fields
	if msg.ID == 0 {
		logrus.Error("Message ID is required to create message")
		return errors.New("message ID is required")
	}
	if msg.Content == "" && msg.FileURL == nil {
		logrus.Error("message must have content or file to create message")
		return errors.New("message must have content or file")
	}
	if msg.UserID == 0 {
		logrus.Error("user ID is required to create message")
		return errors.New("user ID is required")
	}
	if msg.ConversationID == 0 {
		logrus.Error("Conversation ID is required to create message")
		return errors.New("conversation ID is required")
	}

	// Insert into database
	_, err := database.MongoDB.Collection("direct_messages").InsertOne(ctx, msg)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"content_length":  len(msg.Content),
			"file_url":        msg.FileURL,
			"user_id":         msg.UserID,
			"conversation_id": msg.ConversationID,
		}).WithError(err).Error("Failed to insert message in direct messages")
		return errors.New("Failed to insert the message in direct messages")
	}

	return nil
}
