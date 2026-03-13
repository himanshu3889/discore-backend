package channelMessageStore

import (
	"context"

	"github.com/himanshu3889/discore-backend/base/databases"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	"github.com/himanshu3889/discore-backend/base/models"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sirupsen/logrus"
)

// Create message in the database
func CreateChannelMessage(ctx context.Context, msg *models.ChannelMessage) (*models.ChannelMessage, *appError.Error) {
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
		return nil, appError.NewBadRequest("message ID is required")
	}
	if msg.Content == "" && msg.FileURL == nil {
		logrus.Error("message must have content or file to create message")
		return nil, appError.NewBadRequest("message must have content or file")
	}
	if msg.ServerID == 0 {
		logrus.Error("Server ID is required to create message")
		return nil, appError.NewBadRequest("server ID is required")
	}
	if msg.ChannelID == 0 {
		logrus.Error("channel ID is required to create message")
		return nil, appError.NewBadRequest("channel ID is required")
	}
	if msg.UserID == 0 {
		logrus.Error("user ID is required to create message")
		return nil, appError.NewBadRequest("user ID is required")
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
		return nil, appError.NewInternal("Failed to insert the message in channel messages")
	}
	return msg, nil
}

// Insert the bulk channel messages in db
func CreateChannelMessagesBulk(ctx context.Context, msgs []*models.ChannelMessage) (failedMsgIndices []int, appErr *appError.Error) {
	if len(msgs) == 0 {
		return nil, nil
	}

	var validDocuments []interface{}
	var validToOriginalIndex []int

	for i, msg := range msgs {
		deleted := false
		msg.Deleted = &deleted

		// Validate required fields
		if msg.ID == 0 || (msg.Content == "" && msg.FileURL == nil) || msg.ServerID == 0 || msg.ChannelID == 0 || msg.UserID == 0 {
			// Record the original index of the invalid message
			failedMsgIndices = append(failedMsgIndices, i)
			continue
		}

		validDocuments = append(validDocuments, msg)
		validToOriginalIndex = append(validToOriginalIndex, i) // Track the original index
	}

	// If no valid documents, return the failed indices immediately
	if len(validDocuments) == 0 {
		return failedMsgIndices, nil
	}

	opts := options.InsertMany().SetOrdered(false)
	_, err := database.MongoDB.Collection("channel_messages").InsertMany(ctx, validDocuments, opts)

	if err != nil {
		if bulkErr, ok := err.(mongo.BulkWriteException); ok {
			logrus.Warnf("Partial DB insert: %d messages failed", len(bulkErr.WriteErrors))
			// Match the MongoDB errors back to the original msgs index
			for _, writeErr := range bulkErr.WriteErrors {
				originalIndex := validToOriginalIndex[writeErr.Index]
				failedMsgIndices = append(failedMsgIndices, originalIndex)
			}
			return failedMsgIndices, nil
		}

		logrus.WithError(err).Error("Fatal database error on bulk message create")
		return failedMsgIndices, appError.NewInternal("Database bulk message insert error")
	}

	return failedMsgIndices, nil
}
