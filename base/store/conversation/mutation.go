package conversation

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/himanshu3889/discore-backend/base/databases"
	"github.com/himanshu3889/discore-backend/base/models"
	"github.com/himanshu3889/discore-backend/base/utils"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Create conversation b/w two users
func GetOrCreateConversation(ctx context.Context, user1ID, user2ID snowflake.ID) (*models.Conversation, error) {
	if user1ID == 0 {
		logrus.Error("User1 ID is required to create message")
		return nil, errors.New("user1 ID is required")
	}
	if user2ID == 0 {
		logrus.Error("User2 ID is required to create message")
		return nil, errors.New("user2 ID is required")
	}
	if user1ID == user2ID {
		logrus.Error("Self conversations not allowed")
		return nil, errors.New("Self conversations not allowed")
	}

	// var conversation = &models.Conv
	createdAt := time.Now()
	conversationID := utils.GenerateSnowflakeID()
	conversation := models.Conversation{
		ID:        conversationID,
		User1ID:   user1ID,
		User2ID:   user2ID,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	// Maintain sorting in user_ids for uniqueness user1ID < user2ID
	if conversation.User1ID > conversation.User2ID {
		conversation.User1ID, conversation.User2ID = conversation.User2ID, conversation.User1ID
	}

	// query := `
	//     INSERT INTO conversations (id, user1_id, user2_id, created_at, updated_at)
	//     VALUES ($1, $2, $3, $4, $5)
	//     ON CONFLICT (user1_id, user2_id) DO UPDATE SET
	//         id = conversations.id
	//     RETURNING *
	// `

	query := `
        WITH inserted AS (
            INSERT INTO conversations (id, user1_id, user2_id, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (user1_id, user2_id) DO UPDATE SET
                id = conversations.id
            RETURNING id, user1_id, user2_id, created_at, updated_at
        )
        SELECT 
            i.id, i.user1_id, i.user2_id, i.created_at, i.updated_at,
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
        FROM inserted i
        JOIN users u1 ON u1.id = i.user1_id
        JOIN users u2 ON u2.id = i.user2_id
    `

	// Insert into database
	err := database.PostgresDB.GetContext(ctx, &conversation, query,
		conversation.ID,
		conversation.User1ID,
		conversation.User2ID,
		conversation.CreatedAt,
		conversation.UpdatedAt,
	)
	if err != nil {
		if utils.IsDBUniqueViolationError(err) {
			logrus.WithFields(logrus.Fields{"user1_id": conversation.User1ID, "user2_id": conversation.User2ID}).Warn("users conversation already exists in database")
			return nil, errors.New("users conversation already exists in database")
		}
		logrus.WithFields(logrus.Fields{
			"user1_id": conversation.User1ID,
			"user2_id": conversation.User2ID,
		}).WithError(err).Error("Failed to insert message in direct messages")
		return nil, errors.New("Failed to insert the message in direct messages")
	}
	conversation.MeID = &user1ID
	return &conversation, nil

}

// Updates the updated_at field for a conversation
func UpdateConversationTimestampForUser(ctx context.Context, conversationID, userID snowflake.ID) (*models.Conversation, error) {
	if conversationID == 0 || userID == 0 {
		logrus.Error("Conversation ID and User ID are required")
		return nil, errors.New("conversation ID and user ID are required")
	}

	query := `
		UPDATE conversations
		SET updated_at = NOW()
		WHERE id = $1 
		  AND (user1_id = $2 OR user2_id = $2)
		RETURNING *
	`

	var conversation models.Conversation
	err := database.PostgresDB.GetContext(ctx, &conversation, query, conversationID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No conversation found
		}
		logrus.WithFields(logrus.Fields{
			"conversation_id": conversationID,
		}).WithError(err).Error("Failed to update conversation timestamp")
		return nil, errors.New("failed to update conversation")
	}

	return &conversation, nil
}

// Create conversation b/w two users
func CreateConversation(ctx context.Context, conversation *models.Conversation) (*models.Conversation, error) {
	if conversation.User1ID == 0 {
		logrus.Error("User1 ID is required to create message")
		return nil, errors.New("user1 ID is required")
	}
	if conversation.User2ID == 0 {
		logrus.Error("User2 ID is required to create message")
		return nil, errors.New("user2 ID is required")
	}

	conversation.CreatedAt = time.Now()
	conversation.ID = utils.GenerateSnowflakeID()

	// Maintain sorting in user_ids for uniqueness user1ID < user2ID
	if conversation.User1ID > conversation.User2ID {
		conversation.User1ID, conversation.User2ID = conversation.User2ID, conversation.User1ID
	}

	query := `
			INSERT INTO conversations
			(id, user1_id, user2_id, created_at)
			VALUES ($1, $2, $3, $4)
			RETURNING *			
			`

	// Insert into database
	err := database.PostgresDB.GetContext(ctx, conversation, query,
		conversation.ID,
		conversation.User1ID,
		conversation.User2ID,
		conversation.CreatedAt,
	)
	if err != nil {
		if utils.IsDBUniqueViolationError(err) {
			logrus.WithFields(logrus.Fields{"user1_id": conversation.User1ID, "user2_id": conversation.User2ID}).Warn("users conversation already exists in database")
			return nil, errors.New("users conversation already exists in database")
		}
		logrus.WithFields(logrus.Fields{
			"user1_id": conversation.User1ID,
			"user2_id": conversation.User2ID,
		}).WithError(err).Error("Failed to insert message in direct messages")
		return nil, errors.New("Failed to insert the message in direct messages")
	}

	return conversation, nil
}
