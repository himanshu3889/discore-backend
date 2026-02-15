package channelCacheStore

import (
	redisDatabase "discore/internal/base/infrastructure/redis"
	"discore/internal/base/infrastructure/redis/bloomFilter"
	rediskeys "discore/internal/base/lib/redisKeys"
	"discore/internal/modules/core/models"
	channelStore "discore/internal/modules/core/store/channel"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
)

func GetChannelByID(ctx *gin.Context, channelID snowflake.ID) (*models.Channel, error) {
	channelCacheKey, cacheBoundedKey := rediskeys.Keys.Channel.Info(channelID)
	channelBloomKey := bloomFilter.ChannelIDBloomFilter
	bloomItem := channelID.String()
	serverBytes, _ := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, channelCacheKey, &channelBloomKey, &bloomItem)
	if serverBytes == nil {
		// Channel does not exist
		return nil, fmt.Errorf("Channel does not exist")
	}
	return channelStore.GetChannelByID(ctx, channelID)
}
