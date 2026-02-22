package serverCacheStore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	memberCacheStore "github.com/himanshu3889/discore-backend/base/cacheStore/member"
	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	modelsLib "github.com/himanshu3889/discore-backend/base/lib/models"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	"github.com/himanshu3889/discore-backend/base/models"
	serverStore "github.com/himanshu3889/discore-backend/base/store/server"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

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
	serverInviteBloomKey := bloomFilter.ServerInviteBloomFilter
	bloomItem := serverInvite.Code
	codeUsageKey, _ := rediskeys.Keys.ServerInvite.UsedCount(serverInvite.Code)
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, serverInviteCacheKey, &serverInviteBloomKey, serverInvite, &bloomItem, 14*24*time.Hour)
		if serverInvite.MaxUses != nil || *serverInvite.MaxUses <= 0 {
			redisDatabase.GlobalCacheManager.Set(ctx, codeUsageKey, &serverInviteBloomKey, serverInvite.UsedCount, &bloomItem, 14*24*time.Hour)
		}
	}()

	return nil
}

var inviteDbFlightGroup singleflight.Group

// Get the server invite with singleflight; avoid thunderherd
func GetServerInvite(ctx context.Context, code string) (*models.ServerInvite, error) {
	// Check DB, how you deal with the usedCounts
	res, err, _ := inviteDbFlightGroup.Do(code, func() (interface{}, error) {
		serverInviteCacheKey, _ := rediskeys.Keys.ServerInvite.Info(code)
		serverInviteBloomKey := bloomFilter.ServerInviteBloomFilter
		bloomItem := code
		serverInvite, err := serverStore.GetServerInvite(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("Invite code error") // TODO: EXACT ERROR ?, not found, db error ?
		}
		// Save to cache
		redisDatabase.GlobalCacheManager.Set(
			ctx,
			serverInviteCacheKey,
			&serverInviteBloomKey,
			serverInvite,
			&bloomItem,
			24*time.Hour,
		)
		// Update the usage of the code
		codeUsageKey, _ := rediskeys.Keys.ServerInvite.UsedCount(serverInvite.Code)
		redisDatabase.GlobalCacheManager.Set(ctx, codeUsageKey, &serverInviteBloomKey, serverInvite.UsedCount, &bloomItem, 25*time.Hour)
		return serverInvite, nil
	})
	if err != nil {
		return nil, err
	}

	serverInvite := res.(*models.ServerInvite)
	return serverInvite, nil
}

// Accept the server invite and create memember; if already a member then don't consume invite, return serverInvite
// TODO: Require logging
func AcceptServerInviteAndCreateMember(ctx context.Context, userID snowflake.ID, code string) (*models.ServerInvite, error) {
	// Check in cache first then accept
	serverInviteCacheKey, cacheBoundedKey := rediskeys.Keys.ServerInvite.Info(code)
	serverInviteBloomKey := bloomFilter.ServerInviteBloomFilter
	serverInviteBloomItem := code
	codeBytes, cacheErr := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, serverInviteCacheKey, &serverInviteBloomKey, &serverInviteBloomItem)
	if codeBytes == nil && cacheErr == nil {
		// Server invite does not exist
		return nil, fmt.Errorf("Invalid code")
	}

	serverInvite := &models.ServerInvite{}
	var err error
	if cacheErr == nil {
		// If no cache error then unmarshall the cache bytes
		err = json.Unmarshal(codeBytes, serverInvite)
		if err != nil {
			return nil, err
		}
	} else {
		// If cache miss then load and cache it first
		serverInvite, err = GetServerInvite(ctx, code)
		if err != nil {
			return nil, err
		}
	}

	// Code usage check; Fail fast by usage limit; if not unlimited usage
	err = modelsLib.ValidateServerInvite(serverInvite)
	if err != nil {
		return nil, err
	}

	// Check if already the member of the server or not ?
	hasAlreadyMember, err := memberCacheStore.HasUserServerMember(ctx, userID, serverInvite.ServerID)

	if err != nil {
		return nil, err
	}
	if hasAlreadyMember {
		return nil, fmt.Errorf("Already a member of the server")
	}

	// Let's try to create member and accept invite

	// FIXME: lost it if failed; Improve this using kafka; first push in kafka; reduce redis count
	// Use the invite
	err = consumeServerInviteCache(ctx, serverInvite)
	if err != nil {
		return nil, err
	}

	// Create member
	_, err = serverStore.CreateServerMember(ctx, userID, serverInvite.ServerID)
	if err != nil {
		return nil, err
	}

	// Could combine multiples to update the count
	go func() {
		serverStore.UseServerInvite(ctx, serverInvite.Code)
	}()

	return serverInvite, err
}

// Consume the server invite using the cache
var consumeInviteScript = redis.NewScript(`
	local usageKey = KEYS[1]
	local maxUses = tonumber(ARGV[1])
	
	-- Get current uses, default to -1 if key doesn't exist; this thing need to be pre exist
	local currentUses = tonumber(redis.call("GET", usageKey) or "-1")
	if currentUses < 0 then
		return -1 -- Not found in cache
	end
	
	-- If maxUses is greater than 0, enforce the limit
	if maxUses > 0 and currentUses >= maxUses then
		return -429 -- Limit exceeded
	end
	
	-- Increment atomically
	local newCount = redis.call("INCR", usageKey)
	
	return newCount
`)

// Consume the server invite using the redis
func consumeServerInviteCache(ctx context.Context, serverInvite *models.ServerInvite) error {
	// validate invite
	err := modelsLib.ValidateServerInvite(serverInvite)
	if err != nil {
		return err
	}

	// Check if the pointer field is nil before dereferencing
	var maxUsesLimit int64 = -1 // -1 is basically infinite use
	if serverInvite.MaxUses != nil {
		maxUsesLimit = int64(*serverInvite.MaxUses)
	}

	usageKey, boundedKey := rediskeys.Keys.ServerInvite.UsedCount(serverInvite.Code)
	rawResult, err := redisDatabase.GlobalCacheManager.RunScript(
		ctx,
		boundedKey,
		consumeInviteScript,
		[]string{usageKey},
		maxUsesLimit,
	)

	if err != nil {
		return fmt.Errorf("Failed to execute usage script: %w", err)
	}

	newUsedCount, ok := rawResult.(int64)
	if !ok {
		return fmt.Errorf("Unexpected script return type")
	}

	if newUsedCount == -429 {
		return fmt.Errorf("Code usage limit exceeded")
	}

	if newUsedCount == -1 {
		return fmt.Errorf("Cache code usage error")
	}

	// Update the local struct so the caller has the fresh count
	serverInvite.UsedCount = int(newUsedCount)

	return err

}
