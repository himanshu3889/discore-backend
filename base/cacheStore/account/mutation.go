package accountCacheStore

import (
	"context"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	"github.com/himanshu3889/discore-backend/base/models"
	accountStore "github.com/himanshu3889/discore-backend/base/store/account"
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
	cacheKey, _ := rediskeys.Keys.User.Info(user.ID)
	bloomKey := bloomFilter.UserIDBloomFilter
	bloomItem := user.ID.String()

	go func() {
		err = redisDatabase.GlobalCacheManager.Set(ctx, cacheKey, &bloomKey, user, &bloomItem, 0)
		if err != nil {
			// set cache error
		}
	}()

	return nil
}
