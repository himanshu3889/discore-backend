package chatApi

import (
	"net/http"
	"strconv"

	memberCacheStore "github.com/himanshu3889/discore-backend/base/cacheStore/member"
	"github.com/himanshu3889/discore-backend/base/middlewares"
	channelMessageStore "github.com/himanshu3889/discore-backend/base/store/channelMessage"
	"github.com/himanshu3889/discore-backend/base/utils"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
)

func registerChannelMessageRoutes(rg *gin.RouterGroup) {
	chat := rg.Group("/channel")
	channelMessageRoutes(chat)

}

func channelMessageRoutes(rg *gin.RouterGroup) {
	rg.GET("/:channelID/server/:serverID/messages", channelMessages)
}

// Get the channel message
func channelMessages(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	// Get parameter by name
	channelID := ctx.Param("channelID")
	serverID := ctx.Param("serverID")
	channelSnowID, err := utils.ValidSnowflakeID(channelID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid channel ID")
		return
	}
	serverSnowID, err := utils.ValidSnowflakeID(serverID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid server ID")
		return
	}

	// --- GET LIMIT, BEFORE CURSOR FROM QUERY PARAM ---
	limitStr := ctx.DefaultQuery("limit", "50") // Default to 50 if not provided
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Limit must be a positive number")
		return
	}

	var afterCursor *snowflake.ID
	if afterStr := ctx.Query("before"); afterStr != "" {
		afterCursorSnow, err := utils.ValidSnowflakeID(afterStr)
		if err == nil {
			afterCursor = &afterCursorSnow
		}
	}

	memberCacheStore.HasUserServerMember(ctx, userID, serverSnowID)
	messages, err := channelMessageStore.GetServerChannelLastMessages(ctx, serverSnowID, channelSnowID, limit, afterCursor)

	// Validate it's a valid ID
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":  "Chat message fetched",
		"messages": messages,
	})
}
