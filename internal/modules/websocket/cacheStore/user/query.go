package userCacheStore

import (
	"context"
	redisDatabase "discore/internal/base/infrastructure/redis"
	rediskeys "discore/internal/base/lib/redisKeys"
	"discore/internal/modules/chat/models"
	"encoding/json"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// GetUsersBatch fetches users using the cache only
func GetUsersBatch(ctx context.Context, userIDs []snowflake.ID) (map[snowflake.ID]*models.User, error) {
	if len(userIDs) > 100 {
		return nil, errors.New("Max user batching is 100")
	}
	if len(userIDs) == 0 {
		return map[snowflake.ID]*models.User{}, nil
	}

	userMap := make(map[snowflake.ID]*models.User, len(userIDs))

	// Prepare Redis keys
	keys := make([]string, len(userIDs))
	idMap := make(map[string]snowflake.ID, len(userIDs)) // for mapping back
	for i, id := range userIDs {
		key := rediskeys.Keys.User.Info(id)
		keys[i] = key
		idMap[key] = id
	}

	// Bulk fetch from Redis
	cachedData, err := redisDatabase.GlobalCacheManager.MGet(ctx, keys)
	if err != nil {
		logrus.WithError(err).Error("Redis MGet failed")
		// If Redis fails, treat all as missing
	} else {
		for _, raw := range cachedData {
			if raw == nil {
				continue
			}

			// Unmarshal user
			var user models.User
			if err := json.Unmarshal(raw, &user); err != nil {
				// logrus.WithError(err).Error("Unmarshall error")
				continue
			}

			userMap[user.ID] = &user
		}
	}

	return userMap, nil
}
