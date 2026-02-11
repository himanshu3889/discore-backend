package accountCacheStore

import (
	"context"
	redisDatabase "discore/internal/base/infrastructure/redis"
	"discore/internal/base/infrastructure/redis/bloomFilter"
	rediskeys "discore/internal/base/lib/redisKeys"
	"discore/internal/gateway/authenticationService/models"
	accountStore "discore/internal/gateway/authenticationService/store/account"
)

// Create user by write back cache strategy
func CreateUser(ctx context.Context, user *models.User) error {
	// Write Back cache

	// DB
	err := accountStore.CreateUser(ctx, user)
	if err != nil {
		return err
	}

	// async Set cache
	cacheKey := rediskeys.Keys.User.Info(user.ID)
	bloomKey := bloomFilter.UserIDBloomFilter

	go func() {
		err = redisDatabase.GlobalCacheManager.Set(ctx, cacheKey, &bloomKey, user, 0)
		if err != nil {
			// set cache error
		}
	}()

	return nil
}
