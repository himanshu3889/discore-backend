package conversation

import (
	"context"
	"database/sql"
	"discore/internal/modules/chat/database"
	"discore/internal/modules/chat/models"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Gets a conversation only if the user is a participant
func GetConversationForUser(ctx context.Context, conversationID, userID snowflake.ID) (*models.Conversation, error) {
	if conversationID == 0 || userID == 0 {
		logrus.Error("Conversation ID and User ID are required")
		return nil, errors.New("conversation ID and user ID are required")
	}

	query := `
		SELECT *
		FROM conversations
		WHERE id = $1 
		  AND (user1_id = $2 OR user2_id = $2)
	`

	var conversation models.Conversation
	err := database.PostgresDB.GetContext(ctx, &conversation, query, conversationID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not a participant or doesn't exist
		}
		logrus.WithFields(logrus.Fields{
			"conversation_id": conversationID,
			"user_id":         userID,
		}).WithError(err).Error("Failed to fetch conversation for user")
		return nil, errors.New("failed to fetch conversation")
	}

	return &conversation, nil
}

// Gets a conversation between two specific users
func GetConversationBetweenUsers(ctx context.Context, user1ID, user2ID snowflake.ID) (*models.Conversation, error) {
	if user1ID == 0 || user2ID == 0 {
		logrus.Error("Both user IDs are required to fetch conversation")
		return nil, errors.New("both user IDs are required")
	}

	// Enforce ID ordering to match CHECK constraint
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	query := `
		SELECT *
		FROM conversations
		WHERE user1_id = $1 AND user2_id = $2
		LIMIT 1
	`

	var conversation models.Conversation
	err := database.PostgresDB.GetContext(ctx, &conversation, query,
		user1ID,
		user2ID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No conversation exists
		}
		logrus.WithFields(logrus.Fields{
			"user1_id": user1ID,
			"user2_id": user2ID,
		}).WithError(err).Error("Failed to fetch conversation between users")
		return nil, errors.New("failed to fetch conversation between users")
	}

	return &conversation, nil
}

// Gets all conversations for a specific user // FIXME: Need to fetch more ?
func GetAllConversationsForUser(ctx context.Context, userID snowflake.ID, limit int64) ([]models.Conversation, error) {
	if userID == 0 {
		logrus.Error("User ID is required to fetch conversations")
		return nil, errors.New("user ID is required")
	}

	if limit <= 0 {
		limit = 50 // Default limit
	}

	// query := `
	// 	SELECT *
	// 	FROM conversations
	// 	WHERE user1_id = $1 OR user2_id = $1
	// 	ORDER BY updated_at DESC
	// 	LIMIT $2
	// `

	query := `
        SELECT 
            c.id, c.user1_id, c.user2_id, c.created_at, c.updated_at,
            u1.id as "user1.id", 
            u1.username as "user1.username", 
            u1.email as "user1.email",
            u1.name as "user1.name",
            u1.image_url as "user1.image_url",
            u2.id as "user2.id", 
            u2.username as "user2.username", 
            u2.email as "user2.email",
            u2.name as "user2.name",
            u2.image_url as "user2.image_url"
        FROM conversations c
        JOIN users u1 ON u1.id = c.user1_id
        JOIN users u2 ON u2.id = c.user2_id
        WHERE c.user1_id = $1 OR c.user2_id = $1
        ORDER BY c.updated_at DESC
        LIMIT $2
    `

	var conversations []models.Conversation
	err := database.PostgresDB.SelectContext(ctx, &conversations, query,
		userID,
		limit)

	for i := range conversations {
		conversations[i].MeID = &userID
	}

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id": userID,
			"limit":   limit,
		}).WithError(err).Error("Failed to fetch user's conversations")
		return nil, errors.New("failed to fetch user's conversations")
	}

	return conversations, nil
}
