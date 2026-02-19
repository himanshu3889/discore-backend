package memberCacheStore

import (
	"context"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	serverStore "github.com/himanshu3889/discore-backend/base/store/server"

	"github.com/bwmarrin/snowflake"
)

// Has user member of server; check the server in cache
func HasUserServerMember(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (bool, error) {
	return true, nil
	// Validate the server
	cacheKey, cacheBoundedKey := rediskeys.Keys.Server.Info(serverID)
	bloomKey := bloomFilter.ServerIDBloomFilter
	bloomItem := serverID.String()
	serverBytes, _ := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, cacheKey, &bloomKey, &bloomItem)

	if serverBytes == nil {
		// Server does not exist
		return false, nil
	}

	return serverStore.HasUserServerMember(ctx, userID, serverID)
}
