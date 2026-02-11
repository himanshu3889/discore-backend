package directmessageStore

import (
	"context"
	"database/sql"
	"discore/internal/modules/websocket/database"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// conversation is valid only if the user is a participant
func HasValidConversationForUser(ctx context.Context, conversationID, userID snowflake.ID) (bool, error) {
	if conversationID == 0 || userID == 0 {
		logrus.Error("Conversation ID and User ID are required")
		return false, errors.New("conversation ID and user ID are required")
	}

	query := `
		SELECT EXISTS(SELECT 1
		FROM conversations
		WHERE id = $1 
		  AND (user1_id = $2 OR user2_id = $2)
		)
	`

	var valid bool
	err := database.PostgresDB.GetContext(ctx, &valid, query, conversationID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // Not a participant or doesn't exist
		}
		logrus.WithFields(logrus.Fields{
			"conversation_id": conversationID,
			"user_id":         userID,
		}).WithError(err).Error("Failed to fetch conversation for user")
		return false, errors.New("failed to fetch conversation")
	}

	return valid, nil
}
