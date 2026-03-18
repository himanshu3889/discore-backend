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
	serverInviteLib "github.com/himanshu3889/discore-backend/base/lib/serverInvite"
	"github.com/himanshu3889/discore-backend/base/models"
	serverStore "github.com/himanshu3889/discore-backend/base/store/server"
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

	// Use the invite
	appErr = serverInviteLib.ConsumeServerInviteCache(ctx, serverInvite)
	if appErr != nil {
		return nil, appErr
	}

	// Create member
	_, appErr = serverStore.CreateServerMember(ctx, userID, serverInvite.ServerID, &code)
	if appErr != nil {
		// Error creating the member rollback the consume invite
		serverInviteLib.RollbackConsumeServerInviteCache(ctx, serverInvite)
		return nil, appErr
	}

	return serverInvite, appErr
}
