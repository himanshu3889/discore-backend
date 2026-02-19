package channelCacheStore

import (
	"context"
	"time"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	"github.com/himanshu3889/discore-backend/base/models"
	channelStore "github.com/himanshu3889/discore-backend/base/store/channel"

	"github.com/bwmarrin/snowflake"
)

// Create channel and write around cache
func CreateChannel(ctx context.Context, channel *models.Channel) error {
	// DB creation
	err := channelStore.CreateChannel(ctx, channel)
	if err != nil {
		return err
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
func UpdateChannelNameType(ctx context.Context, channel *models.Channel) error {
	// DB creation
	err := channelStore.UpdateChannelNameType(ctx, channel)
	if err != nil {
		return err
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
func HardDeleteChannelById(ctx context.Context, channelID snowflake.ID) (*models.Channel, error) {
	// DB creation
	channel, err := channelStore.HardDeleteChannelById(ctx, channelID)
	if err != nil {
		return nil, err
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
