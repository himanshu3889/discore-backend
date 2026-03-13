package serverInviteLib

import (
	"context"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	modelsLib "github.com/himanshu3889/discore-backend/base/lib/models"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	"github.com/himanshu3889/discore-backend/base/models"
)

// Consume the server invite using the redis
func ConsumeServerInviteCache(ctx context.Context, serverInvite *models.ServerInvite) *appError.Error {
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

// Rollback the consumer server Invite of redis
func RollbackConsumeServerInviteCache(ctx context.Context, serverInvite *models.ServerInvite) *appError.Error {
	// validate invite
	usageKey, boundedKey := rediskeys.Keys.ServerInvite.UsedCount(serverInvite.Code)
	rawResult, err := redisDatabase.GlobalCacheManager.RunScript(
		ctx,
		boundedKey,
		rollbackInviteScript,
		[]string{usageKey},
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
