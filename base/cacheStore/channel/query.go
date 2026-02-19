package channelCacheStore

import (
	"fmt"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/infrastructure/redis/bloomFilter"
	rediskeys "github.com/himanshu3889/discore-backend/base/lib/redisKeys"
	"github.com/himanshu3889/discore-backend/base/models"
	channelStore "github.com/himanshu3889/discore-backend/base/store/channel"

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
