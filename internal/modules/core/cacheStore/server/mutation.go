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
	serverCacheKey := rediskeys.Keys.Server.Info(server.ID)
	serverBloomKey := bloomFilter.ServerIDBloomFilter
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, serverCacheKey, &serverBloomKey, server, 14*24*time.Hour)
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
	serverCacheKey := rediskeys.Keys.Server.Info(server.ID)
	serverBloomKey := bloomFilter.ServerIDBloomFilter
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, serverCacheKey, &serverBloomKey, server, 14*24*time.Hour)
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
	serverInviteCacheKey := rediskeys.Keys.ServerInvite.Info(serverInvite.Code)
	serverBloomKey := bloomFilter.ServerInviteBloomFilter
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, serverInviteCacheKey, &serverBloomKey, serverInvite, 14*24*time.Hour)
	}()

	return nil
}

// Accept the server invite and create memember; if already a member then don't consume invite, return serverInvite
func AcceptServerInviteAndCreateMember(ctx context.Context, userID snowflake.ID, code string) (*models.ServerInvite, error) {
	// Check in cache first then accept
	serverInviteCacheKey := rediskeys.Keys.ServerInvite.Info(code)
	serverInviteBloomKey := bloomFilter.ServerInviteBloomFilter
	codeBytes, cacheErr := redisDatabase.GlobalCacheManager.Get(ctx, serverInviteCacheKey, &serverInviteBloomKey)
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
			24*time.Hour,
		)
	}

	return serverInvite, err
}
