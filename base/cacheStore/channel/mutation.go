package channelCacheStore

import (
	"context"
	"time"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	"github.com/himanshu3889/discore-backend/base/models"
	channelStore "github.com/himanshu3889/discore-backend/base/store/channel"

	"github.com/bwmarrin/snowflake"
)

// Create channel and write around cache
func CreateChannel(ctx context.Context, channel *models.Channel) *appError.Error {
	// DB creation
	appErr := channelStore.CreateChannel(ctx, channel)
	if appErr != nil {
		return appErr
	}

	// async write to cache
	channelCacheKey, _ := rediskeys.Keys.Channel.Info(channel.ID)
	channelBloomKey := bloomFilter.ChannelIDBloomFilter
	bloomItem := channel.ID.String()
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, channelCacheKey, &channelBloomKey, channel, &bloomItem, 14*24*time.Hour)
	}()
	return nil
}

// Update channel by name and type; write around cache
func UpdateChannelNameType(ctx context.Context, channel *models.Channel) *appError.Error {
	// DB creation
	appErr := channelStore.UpdateChannelNameType(ctx, channel)
	if appErr != nil {
		return appErr
	}

	// async write to cache
	channelCacheKey, _ := rediskeys.Keys.Channel.Info(channel.ID)
	channelBloomKey := bloomFilter.ChannelIDBloomFilter
	bloomItem := channel.ID.String()
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, channelCacheKey, &channelBloomKey, channel, &bloomItem, 14*24*time.Hour)
	}()
	return nil
}

// Hard delete channel by id and write around it cache set null
func HardDeleteChannelById(ctx context.Context, channelID snowflake.ID) (*models.Channel, *appError.Error) {
	// DB creation
	channel, appErr := channelStore.HardDeleteChannelById(ctx, channelID)
	if appErr != nil {
		return nil, appErr
	}

	// async write to cache
	channelCacheKey, _ := rediskeys.Keys.Channel.Info(channel.ID)
	channelBloomKey := bloomFilter.ChannelIDBloomFilter
	bloomItem := channel.ID.String()
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, channelCacheKey, &channelBloomKey, nil, &bloomItem, 2*24*time.Hour)
	}()
	return channel, nil
}
