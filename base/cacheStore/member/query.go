package memberCacheStore

import (
	"context"
	"time"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	serverStore "github.com/himanshu3889/discore-backend/base/store/server"

	"github.com/bwmarrin/snowflake"
)

// Has user member of server; check the server in cache
func HasUserServerMember(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (bool, *appError.Error) {
	// Validate the server
	cacheKey, cacheBoundedKey := rediskeys.Keys.Server.Info(serverID)
	bloomKey := bloomFilter.ServerIDBloomFilter
	bloomItem := serverID.String()
	serverBytes, err := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, cacheKey, &bloomKey, &bloomItem)
	if serverBytes == nil && err == nil {
		// Server does not exists
		return false, nil
	}

	if err != nil {
		server, appErr := serverStore.GetServerByID(ctx, serverID)
		if appErr != nil {
			return false, appErr
		}
		redisDatabase.GlobalCacheManager.Set(ctx, cacheKey, &bloomKey, server, &bloomItem, 30*24*time.Hour)
	}

	return serverStore.HasUserServerMember(ctx, userID, serverID)
}
