package channelCacheStore

import (
	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	"github.com/himanshu3889/discore-backend/base/models"
	channelStore "github.com/himanshu3889/discore-backend/base/store/channel"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
)

// Get channel by ID; using cache
func GetChannelByID(ctx *gin.Context, channelID snowflake.ID) (*models.Channel, *appError.Error) {
	channelCacheKey, cacheBoundedKey := rediskeys.Keys.Channel.Info(channelID)
	channelBloomKey := bloomFilter.ChannelIDBloomFilter
	bloomItem := channelID.String()
	serverBytes, _ := redisDatabase.GlobalCacheManager.Get(ctx, cacheBoundedKey, channelCacheKey, &channelBloomKey, &bloomItem)
	if serverBytes == nil {
		// Channel does not exist
		return nil, appError.NewNotFound("Channel does not exist")
	}
	return channelStore.GetChannelByID(ctx, channelID)
}
