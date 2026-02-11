package directMessageStore

import (
	"context"
	"discore/internal/modules/chat/database"
	"discore/internal/modules/chat/models"
	userStore "discore/internal/modules/chat/store/user"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Get channel last messages
func GetConversationLastMessages(ctx context.Context, conversationID snowflake.ID, limit int64, afterID *snowflake.ID) ([]*models.DirectMessage, error) {
	// Cap the limit
	if limit > 100 {
		limit = 100
	}

	filter := bson.M{
		"conversation_id": conversationID,
		"deleted":         false,
	}

	// If you have a cursor, get messages before it
	if afterID != nil {
		filter["_id"] = bson.M{"$lt": afterID}
	}

	opts := options.Find()
	opts.SetSort(bson.D{{"_id", -1}})
	opts.SetLimit(limit)

	cursor, err := database.MongoDB.Collection("direct_messages").Find(ctx, filter, opts)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"conversation_id": conversationID,
			"limit":           limit,
		}).WithError(err).Error("Failed to fetch direct messages from database")
		return nil, err
	}

	defer cursor.Close(ctx)

	var messages []*models.DirectMessage
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		messages = []*models.DirectMessage{}
	}

	// Extract unique user IDs
	userIDSet := make(map[snowflake.ID]bool)
	for _, msg := range messages {
		userIDSet[msg.UserID] = true
	}

	userIDs := make([]snowflake.ID, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	// Batch fetch users
	usersMap, err := userStore.GetUsersBatch(ctx, userIDs)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"conversation_id": conversationID,
			"user_ids":        userIDs,
		}).WithError(err).Warn("Failed to fetch users batch - messages will not include author data")
		// Continue without authors rather than failing completely
	}

	// Attach author to each message
	for _, msg := range messages {
		if usersMap != nil {
			msg.User = usersMap[msg.UserID]
		}
	}

	return messages, nil
}
