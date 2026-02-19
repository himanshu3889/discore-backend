package redisDatabase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/discore-backend/configs"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var (
	RedisClient *redis.Client
	redisOnce   sync.Once
)

// Initialize redis
func InitRedis() {
	redisOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		RedisClient = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s",
				configs.Config.REDIS_HOST,
				configs.Config.REDIS_PORT),
			Password:     configs.Config.REDIS_PASSWORD,
			DB:           0,
			PoolSize:     100,
			MinIdleConns: 10,
		})

		// Test connection
		if err := RedisClient.Ping(ctx).Err(); err != nil {
			logrus.WithError(err).Fatal("Redis connection failed")
		}

		logrus.Info("Redis database connected successfully")

		// Initialize others
		InitGlobalBloomManager()
		InitGlobalCacheManager()
	})
}

var bloomOnce sync.Once
var GlobalBloom *bloomFilter.BloomManager

func InitGlobalBloomManager() {
	bloomOnce.Do(func() {
		configs := []bloomFilter.FilterConfig{
			{
				Key:      bloomFilter.UserIDBloomFilter,
				Capacity: 1_000_000,
				FPRate:   0.01,
			},
			{
				Key:      bloomFilter.ServerIDBloomFilter,
				Capacity: 1_000_000,
				FPRate:   0.01,
			},
			{
				Key:      bloomFilter.ChannelIDBloomFilter,
				Capacity: 10_000_000,
				FPRate:   0.01,
			},
			{
				Key:      bloomFilter.ServerInviteBloomFilter,
				Capacity: 10_000_000,
				FPRate:   0.01,
			},
		}

		var err error
		GlobalBloom, err = bloomFilter.NewBloomManager(RedisClient, configs)
		if err != nil {
			logrus.WithError(err).Fatal("unable to initialize global bloom filter")
		}
		logrus.Info("Global bloom filter initialized")
	})
}

var cacheManagerOnce sync.Once
var GlobalCacheManager *CacheManager

func InitGlobalCacheManager() {
	cacheManagerOnce.Do(func() {
		GlobalCacheManager = NewCacheManager(RedisClient, GlobalBloom)
		logrus.Info("Global cache initialized")
	})
}
