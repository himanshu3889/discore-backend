package memberCacheStore

import (
	"context"
	redisDatabase "discore/internal/base/infrastructure/redis"
	"discore/internal/base/infrastructure/redis/bloomFilter"
	rediskeys "discore/internal/base/lib/redisKeys"
	serverStore "discore/internal/modules/websocket/store/server"

	"github.com/bwmarrin/snowflake"
)

// Has user member of server; check the server in cache
func HasUserServerMember(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (bool, error) {

	// Validate the server
	cacheKey := rediskeys.Keys.Server.Info(serverID)
	bloomKey := bloomFilter.ServerIDBloomFilter
	serverBytes, _ := redisDatabase.GlobalCacheManager.Get(ctx, cacheKey, &bloomKey)

	if serverBytes == nil {
		// Server does not exist
		return false, nil
	}

	return serverStore.HasUserServerMember(ctx, userID, serverID)
}
