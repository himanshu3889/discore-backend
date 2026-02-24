package serverCacheStore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	memberCacheStore "github.com/himanshu3889/discore-backend/base/cacheStore/member"
	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	modelsLib "github.com/himanshu3889/discore-backend/base/lib/models"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	"github.com/himanshu3889/discore-backend/base/models"
	serverStore "github.com/himanshu3889/discore-backend/base/store/server"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"

	"github.com/bwmarrin/snowflake"
)

// Create new server and write around cache
func CreateServer(ctx context.Context, server *models.Server) *appError.Error {

	// DB creation
	appErr := serverStore.CreateServer(ctx, server)
	if appErr != nil {
		return appErr
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
func UpdateServerNameImage(ctx context.Context, server *models.Server) *appError.Error {
	// DB creation
	appErr := serverStore.UpdateServerNameImage(ctx, server)
	if appErr != nil {
		return appErr
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
func CreateServerInvite(ctx context.Context, serverInvite *models.ServerInvite) *appError.Error {
	// DB creation
	appErr := serverStore.CreateServerInvite(ctx, serverInvite)
	if appErr != nil {
		return appErr
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
func GetServerInvite(ctx context.Context, code string) (*models.ServerInvite, *appError.Error) {
	// Check DB, how you deal with the usedCounts
	res, err, _ := inviteDbFlightGroup.Do(code, func() (interface{}, error) {
		serverInviteCacheKey, _ := rediskeys.Keys.ServerInvite.Info(code)
		serverInviteBloomKey := bloomFilter.ServerInviteBloomFilter
		bloomItem := code
		serverInvite, appErr := serverStore.GetServerInvite(ctx, code)
		if appErr != nil {
			logrus.WithFields(logrus.Fields{"invite_code": code}).WithError(errors.New(appErr.Message)).Error("Singleflight server invite")
			return appErr, errors.New(appErr.Message) // send the exact err as the res
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
		appErr := res.(*appError.Error) // the exact appErr
		return nil, appErr
	}

	serverInvite := res.(*models.ServerInvite)
	return serverInvite, nil
}

// Accept the server invite and create memember; if already a member then don't consume invite, return serverInvite
// TODO: Require logging
func AcceptServerInviteAndCreateMember(ctx context.Context, userID snowflake.ID, code string) (*models.ServerInvite, *appError.Error) {
	// Check in cache first then accept
	serverInviteCacheKey, cacheBoundedKey := rediskeys.Keys.ServerInvite.Info(code)
	serverInviteBloomKey := bloomFilter.ServerInviteBloomFilter
	serverInviteBloomItem := code
	codeBytes, cacheErr := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, serverInviteCacheKey, &serverInviteBloomKey, &serverInviteBloomItem)
	if codeBytes == nil && cacheErr == nil {
		// Server invite does not exist
		return nil, appError.NewBadRequest("Invalid code")
	}

	serverInvite := &models.ServerInvite{}
	if cacheErr == nil {
		// If no cache error then unmarshall the cache bytes
		err := json.Unmarshal(codeBytes, serverInvite)
		if err != nil {
			logrus.WithFields(logrus.Fields{"invite_code": code}).WithError(err).Error("Unmarshall error in accept server invite")
			return nil, appError.NewInternal(err.Error())
		}
	} else {
		// If cache miss then load and cache it first
		var appErr *appError.Error
		serverInvite, appErr = GetServerInvite(ctx, code)
		if appErr != nil {
			return nil, appErr
		}
	}

	// Code usage check; Fail fast by usage limit; if not unlimited usage
	appErr := modelsLib.ValidateServerInvite(serverInvite)
	if appErr != nil {
		return nil, appErr
	}

	// Check if already the member of the server or not ?
	hasAlreadyMember, appErr := memberCacheStore.HasUserServerMember(ctx, userID, serverInvite.ServerID)

	if appErr != nil {
		return nil, appErr
	}
	if hasAlreadyMember {
		return nil, appError.NewBadRequest("Already a member of the server")
	}

	// Let's try to create member and accept invite

	// FIXME: lost it if failed; Improve this using kafka; first push in kafka; reduce redis count
	// Use the invite
	appErr = consumeServerInviteCache(ctx, serverInvite)
	if appErr != nil {
		return nil, appErr
	}

	// Create member
	_, appErr = serverStore.CreateServerMember(ctx, userID, serverInvite.ServerID)
	if appErr != nil {
		return nil, appErr
	}

	// FIXME: Could combine multiples to update the count
	go func() {
		serverStore.UseServerInvite(ctx, serverInvite.Code)
	}()

	return serverInvite, appErr
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
func consumeServerInviteCache(ctx context.Context, serverInvite *models.ServerInvite) *appError.Error {
	// validate invite
	appErr := modelsLib.ValidateServerInvite(serverInvite)
	if appErr != nil {
		return appErr
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
		return appError.NewInternal(err.Error())
	}

	newUsedCount, ok := rawResult.(int64)
	if !ok {
		return appError.NewInternal("Unexpected script return type")
	}

	if newUsedCount == -429 {
		return &appError.Error{Message: "Code usage limit exceeded", Code: appError.StatusGone}
	}

	if newUsedCount == -1 {
		return appError.NewInternal("Code not found")
	}

	// Update the local struct so the caller has the fresh count
	serverInvite.UsedCount = int(newUsedCount)

	return nil

}
