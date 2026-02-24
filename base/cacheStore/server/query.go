package serverCacheStore

import (
	"context"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	"github.com/himanshu3889/discore-backend/base/models"
	serverStore "github.com/himanshu3889/discore-backend/base/store/server"

	"github.com/bwmarrin/snowflake"
)

func GetServerChannels(ctx context.Context, serverId snowflake.ID) ([]*models.Channel, *appError.Error) {
	serverCacheKey, cacheBoundedKey := rediskeys.Keys.Server.Info(serverId)
	serverBloomKey := bloomFilter.ServerIDBloomFilter
	bloomItem := serverId.String()
	serverBytes, _ := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, serverCacheKey, &serverBloomKey, &bloomItem)
	if serverBytes == nil {
		// Server does not exist
		return nil, appError.NewBadRequest("Server does not exist")
	}
	return serverStore.GetServerChannels(ctx, serverId)

}

func GetServerMembers(ctx context.Context, serverId snowflake.ID, limit int, afterSnowflake snowflake.ID) ([]*models.Member, *appError.Error) {
	serverCacheKey, cacheBoundedKey := rediskeys.Keys.Server.Info(serverId)
	serverBloomKey := bloomFilter.ServerIDBloomFilter
	bloomItem := serverId.String()
	serverBytes, _ := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, serverCacheKey, &serverBloomKey, &bloomItem)
	if serverBytes == nil {
		// Server does not exist
		return nil, appError.NewBadRequest("Server does not exist")
	}
	return serverStore.GetServerMembers(ctx, serverId, limit, afterSnowflake)
}
