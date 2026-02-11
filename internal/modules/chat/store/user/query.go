package userStore

import (
	"context"
	"discore/internal/modules/chat/database"
	"discore/internal/modules/chat/models"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Fetches multiple users in a single query; Limit 100
// TODO: need to implement the caching
func GetUsersBatch(ctx context.Context, userIDs []snowflake.ID) (map[snowflake.ID]*models.User, error) {
	if len(userIDs) > 100 {
		return nil, errors.New("Max user batching is 100")
	}
	if len(userIDs) == 0 {
		return map[snowflake.ID]*models.User{}, nil
	}

	query := `
		SELECT id, username, email, name, image_url
		FROM users
		WHERE id = ANY($1)
	`

	// Convert slice to pq.Array for PostgreSQL
	var users []*models.User
	err := database.PostgresDB.SelectContext(ctx, &users, query, pq.Array(userIDs))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_ids_len": len(userIDs),
		}).WithError(err).Error("Failed to fetch users batch")
		return nil, errors.New("failed to fetch users")
	}

	// Convert to map for O(1) lookups
	userMap := make(map[snowflake.ID]*models.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	return userMap, nil
}
