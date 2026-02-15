package serverCacheStore

import (
	"context"
	serverStore "discore/internal/modules/websocket/store/server"

	"github.com/bwmarrin/snowflake"
)

func HasUserServerMember(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (bool, error) {

	// Validate the server
	// serverCacheKey, cacheBoundedKey := rediskeys.Keys.Server.Info(serverID)
	// bloomKey := bloomFilter.ServerIDBloomFilter
	// bloomItem := serverID.String()
	// serverBytes, _ := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, serverCacheKey, &bloomKey, &bloomItem)

	// if serverBytes == nil {
	// 	// Server does not exist
	// 	return false, nil
	// }

	return serverStore.HasUserServerMember(ctx, userID, serverID)
}
