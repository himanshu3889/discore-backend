package message

import (
	"discore/internal/modules/chat/database"
	"discore/internal/modules/chat/models"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetChannelLastMessages(ctx *gin.Context, channelID snowflake.ID, limit int64, afterID *snowflake.ID) ([]*models.ChannelMessage, error) {
	// Cap the limit
	if limit > 100 {
		limit = 100
	}

	filter := bson.M{
		"channel_id": channelID,
		"deleted":    false,
	}

	// If you have a cursor, get messages before it
	if afterID != nil {
		filter["_id"] = bson.M{"$lt": afterID}
	}

	opts := options.Find()
	opts.SetSort(bson.D{{"created_at", -1}})
	opts.SetLimit(limit)

	cursor, err := database.MongoDB.Collection("channel_messages").Find(ctx, filter, opts)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"channel_id": channelID,
			"limit":      limit,
		}).WithError(err).Error("Failed to fetch messages from database")
		return nil, err
	}

	defer cursor.Close(ctx)

	var messages []*models.ChannelMessage
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		messages = []*models.ChannelMessage{}
	}

	return messages, nil
}
