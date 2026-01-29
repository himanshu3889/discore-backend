package chatApi

import (
	"discore/internal/base/utils"
	"discore/internal/modules/chat/models"
	"discore/internal/modules/chat/store/message"
	"net/http"
	"strconv"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
)

func registerMessageRoutes(rg *gin.RouterGroup) {
	chat := rg.Group("/message")
	messageRoutes(chat)

}

func messageRoutes(rg *gin.RouterGroup) {
	rg.GET("/channel/:channelID", channelMessages)
	rg.POST("/channel/:channelID", channelMessages)
}

// Get the channel message
func channelMessages(ctx *gin.Context) {
	// userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	// if !isOk {
	// 	utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
	// 	return
	// }

	// Get parameter by name
	channelID := ctx.Param("channelID")
	channelSnowID, err := utils.ValidSnowflakeID(channelID)

	// --- GET LIMIT, BEFORE CURSOR FROM QUERY PARAM ---
	limitStr := ctx.DefaultQuery("limit", "50") // Default to 50 if not provided
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Limit must be a positive number")
		return
	}

	var afterCursor *snowflake.ID
	if afterStr := ctx.Query("before"); afterStr != "" {
		channelID := ctx.Param("channelID")
		channelSnowID, err := utils.ValidSnowflakeID(channelID)
		if err == nil {
			afterCursor = &channelSnowID
		}
	}

	messages, err := message.GetChannelLastMessages(ctx, channelSnowID, limit, afterCursor)

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

// Send the message in the channel
func sendChannelMessage(ctx *gin.Context) {
	// userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	// if !isOk {
	// 	utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
	// 	return
	// }

	var incomingMessage models.ChannelMessage
	if err := ctx.ShouldBindJSON(&incomingMessage); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	messages, err := message.CreateChannelMessage(ctx, &incomingMessage)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":  "Chat message fetched",
		"messages": messages,
	})
}
