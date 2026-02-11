package serverCacheStore

import (
	"context"
	redisDatabase "discore/internal/base/infrastructure/redis"
	"discore/internal/base/infrastructure/redis/bloomFilter"
	rediskeys "discore/internal/base/lib/redisKeys"
	"discore/internal/modules/core/models"
	serverStore "discore/internal/modules/core/store/server"
	"fmt"

	"github.com/bwmarrin/snowflake"
)

func GetServerChannels(ctx context.Context, serverId snowflake.ID) ([]*models.Channel, error) {
	serverCacheKey := rediskeys.Keys.Server.Info(serverId)
	serverBloomKey := bloomFilter.ServerIDBloomFilter
	serverBytes, _ := redisDatabase.GlobalCacheManager.Get(ctx, serverCacheKey, &serverBloomKey)
	if serverBytes == nil {
		// Server does not exist
		return nil, fmt.Errorf("Server does not exist")
	}
	return serverStore.GetServerChannels(ctx, serverId)

}

func GetServerMembers(ctx context.Context, serverId snowflake.ID, limit int, afterSnowflake snowflake.ID) ([]*models.Member, error) {
	serverCacheKey := rediskeys.Keys.Server.Info(serverId)
	serverBloomKey := bloomFilter.ServerIDBloomFilter
	serverBytes, _ := redisDatabase.GlobalCacheManager.Get(ctx, serverCacheKey, &serverBloomKey)
	if serverBytes == nil {
		// Server does not exist
		return nil, fmt.Errorf("Server does not exist")
	}
	return serverStore.GetServerMembers(ctx, serverId, limit, afterSnowflake)
}
