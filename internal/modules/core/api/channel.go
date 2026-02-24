package coreApi

import (
	"net/http"

	channelCacheStore "github.com/himanshu3889/discore-backend/base/cacheStore/channel"
	"github.com/himanshu3889/discore-backend/base/middlewares"
	"github.com/himanshu3889/discore-backend/base/models"
	"github.com/himanshu3889/discore-backend/base/utils"

	"github.com/gin-gonic/gin"
)

func registerChannelRoutes(r *gin.RouterGroup) {
	channelGroup := r.Group("/channels")
	channelRoutes(channelGroup)
}

func channelRoutes(rg *gin.RouterGroup) {
	rg.POST("", CreateChannel)
	rg.GET("/:channelID", GetChannelByID)
	rg.PATCH("/:channelID", UpdateChannelByID)
	rg.DELETE("/:channelID", DeleteChannelByID)
}

func CreateChannel(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	var incomingChannel *models.Channel
	if err := ctx.ShouldBindJSON(&incomingChannel); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// Assign the creator id as the user id
	// FIXME: Admin or moderator only can create the channel in the server
	incomingChannel.CreatorID = userID

	appErr := channelCacheStore.CreateChannel(ctx, incomingChannel)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusCreated, gin.H{"channel": incomingChannel, "message": "Channel Created"})
}

func GetChannelByID(ctx *gin.Context) {
	// userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	// if !isOk {
	// 	utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
	// 	return
	// }

	// Get parameter by name
	channelID := ctx.Param("channelID")
	channelSnowID, err := utils.ValidSnowflakeID(channelID)

	// Validate it's a valid ID
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	channel, appErr := channelCacheStore.GetChannelByID(ctx, channelSnowID)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"channel": channel, "message": "Channel found"})
}

func UpdateChannelByID(ctx *gin.Context) {
	// userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	// if !isOk {
	// 	utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
	// 	return
	// }

	// Get parameter by name
	channelID := ctx.Param("channelID")
	channelSnowID, err := utils.ValidSnowflakeID(channelID)

	// Validate it's a valid ID
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	var incomingChannel *models.Channel
	if err := ctx.ShouldBindJSON(&incomingChannel); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	incomingChannel.ID = channelSnowID

	appErr := channelCacheStore.UpdateChannelNameType(ctx, incomingChannel)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"channel": incomingChannel, "message": "Channel found"})
}

func DeleteChannelByID(ctx *gin.Context) {
	// userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	// if !isOk {
	// 	utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
	// 	return
	// }

	// Get parameter by name
	channelID := ctx.Param("channelID")
	channelSnowID, err := utils.ValidSnowflakeID(channelID)

	// Validate it's a valid ID
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	_, appErr := channelCacheStore.HardDeleteChannelById(ctx, channelSnowID)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"channelID": channelID, "message": "Channel deleted successfully"})
}
