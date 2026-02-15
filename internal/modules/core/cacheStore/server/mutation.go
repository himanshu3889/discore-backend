package serverCacheStore

import (
	"context"
	redisDatabase "discore/internal/base/infrastructure/redis"
	"discore/internal/base/infrastructure/redis/bloomFilter"
	rediskeys "discore/internal/base/lib/redisKeys"
	"discore/internal/modules/core/models"
	serverStore "discore/internal/modules/core/store/server"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
)

// Create new server and write around cache
func CreateServer(ctx context.Context, server *models.Server) error {

	// DB creation
	err := serverStore.CreateServer(ctx, server)
	if err != nil {
		return err
	}

	// async write to cache
	serverCacheKey, _ := rediskeys.Keys.Server.Info(server.ID)
	serverBloomKey := bloomFilter.ServerIDBloomFilter
	bloomItem := server.ID.String()
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, serverCacheKey, &serverBloomKey, server, &bloomItem, 14*24*time.Hour)
	}()

	return nil
}

// Update server by name and image; and write around cache
func UpdateServerNameImage(ctx context.Context, server *models.Server) error {
	// DB creation
	err := serverStore.UpdateServerNameImage(ctx, server)
	if err != nil {
		return err
	}

	// async write to cache
	serverCacheKey, _ := rediskeys.Keys.Server.Info(server.ID)
	serverBloomKey := bloomFilter.ServerIDBloomFilter
	bloomItem := server.ID.String()
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, serverCacheKey, &serverBloomKey, server, &bloomItem, 14*24*time.Hour)
	}()

	return nil
}

// Create server invite and write around cache
func CreateServerInvite(ctx context.Context, serverInvite *models.ServerInvite) error {
	// DB creation
	err := serverStore.CreateServerInvite(ctx, serverInvite)
	if err != nil {
		return err
	}

	// async write to cache
	serverInviteCacheKey, _ := rediskeys.Keys.ServerInvite.Info(serverInvite.Code)
	serverBloomKey := bloomFilter.ServerInviteBloomFilter
	bloomItem := serverInvite.Code
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, serverInviteCacheKey, &serverBloomKey, serverInvite, &bloomItem, 14*24*time.Hour)
	}()

	return nil
}

// Accept the server invite and create memember; if already a member then don't consume invite, return serverInvite
func AcceptServerInviteAndCreateMember(ctx context.Context, userID snowflake.ID, code string) (*models.ServerInvite, error) {
	// Check in cache first then accept
	serverInviteCacheKey, cacheBoundedKey := rediskeys.Keys.ServerInvite.Info(code)
	serverInviteBloomKey := bloomFilter.ServerInviteBloomFilter
	bloomItem := code
	codeBytes, cacheErr := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, serverInviteCacheKey, &serverInviteBloomKey, &bloomItem)
	if codeBytes == nil {
		// Server invite does not exist
		return nil, fmt.Errorf("Invalid code")
	}

	// Check DB
	serverInvite, err := serverStore.AcceptServerInviteAndCreateMember(ctx, userID, code)

	// Only attempt to cache if we had a cache miss/error previously
	if cacheErr != nil {
		redisDatabase.GlobalCacheManager.Set(
			ctx,
			serverInviteCacheKey,
			&serverInviteBloomKey,
			serverInvite,
			&bloomItem,
			24*time.Hour,
		)
	}

	return serverInvite, err
}
