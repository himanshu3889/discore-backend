package redisDatabase

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var (
	RedisClient *redis.Client
	once        sync.Once
)

// Initialization happens exactly once
// Thread-safe
// Idempotent

// Initialize redis
func InitRedis() {
	once.Do(func() { // Wrap everything in once.Do
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		RedisClient = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s",
				os.Getenv("REDIS_HOST"),
				os.Getenv("REDIS_PORT")),
			Password:     os.Getenv("REDIS_PASSWORD"),
			DB:           0,
			PoolSize:     100,
			MinIdleConns: 10,
		})

		// Test connection
		if err := RedisClient.Ping(ctx).Err(); err != nil {
			logrus.WithError(err).Fatal("Redis connection failed")
		}

		logrus.Info("Redis database connected successfully")
	})
}
