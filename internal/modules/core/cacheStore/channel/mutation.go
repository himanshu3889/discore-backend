package channelCacheStore

import (
	"context"
	redisDatabase "discore/internal/base/infrastructure/redis"
	"discore/internal/base/infrastructure/redis/bloomFilter"
	rediskeys "discore/internal/base/lib/redisKeys"
	"discore/internal/modules/core/models"
	channelStore "discore/internal/modules/core/store/channel"
	"time"

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
	channelCacheKey := rediskeys.Keys.Channel.Info(channel.ID)
	channelBloomKey := bloomFilter.ChannelIDBloomFilter
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, channelCacheKey, &channelBloomKey, channel, 14*24*time.Hour)
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
	channelCacheKey := rediskeys.Keys.Channel.Info(channel.ID)
	channelBloomKey := bloomFilter.ChannelIDBloomFilter
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, channelCacheKey, &channelBloomKey, channel, 14*24*time.Hour)
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
	channelCacheKey := rediskeys.Keys.Channel.Info(channel.ID)
	channelBloomKey := bloomFilter.ChannelIDBloomFilter
	go func() {
		redisDatabase.GlobalCacheManager.Set(ctx, channelCacheKey, &channelBloomKey, nil, 2*24*time.Hour)
	}()
	return channel, nil
}
