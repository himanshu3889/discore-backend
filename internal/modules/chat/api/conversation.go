package chatApi

import (
	"net/http"
	"strconv"

	"github.com/himanshu3889/discore-backend/base/middlewares"
	conversationStore "github.com/himanshu3889/discore-backend/base/store/conversation"
	directMessageStore "github.com/himanshu3889/discore-backend/base/store/directMessage"
	"github.com/himanshu3889/discore-backend/base/utils"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
)

// Conversation routes
func registerConversationRoutes(rg *gin.RouterGroup) {
	chat := rg.Group("/conversation")
	conversationRoutes(chat)
}

func conversationRoutes(rg *gin.RouterGroup) {
	rg.GET("/:conversationID", getConversationForUser)
	rg.GET("/all", getAllConversationForUser)
	rg.GET("/:conversationID/messages", conversationMessagesForUser)
	rg.POST("/user/:user2ID", getOrCreateConversationForUsers)
}

// Get the conversation for the user
func getConversationForUser(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	// Get parameter by name
	conversationID := ctx.Param("conversationID")
	conversationSnowID, err := utils.ValidSnowflakeID(conversationID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
	}

	conversation, appErr := conversationStore.GetConversationForUser(ctx, conversationSnowID, userID)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":      "Conversation find",
		"conversation": conversation,
	})
}

// Get the conversation for the user
func getAllConversationForUser(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	// --- GET LIMIT, BEFORE CURSOR FROM QUERY PARAM ---
	limitStr := ctx.DefaultQuery("limit", "50") // Default to 50 if not provided
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Limit must be a positive number")
		return
	}

	// var afterCursor *snowflake.ID
	// if afterStr := ctx.Query("before"); afterStr != "" {
	// 	channelID := ctx.Param("channelID")
	// 	channelSnowID, err := utils.ValidSnowflakeID(channelID)
	// 	if err == nil {
	// 		afterCursor = &channelSnowID
	// 	}
	// }

	conversations, appErr := conversationStore.GetAllConversationsForUser(ctx, userID, limit)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":       "Conversation find",
		"conversations": conversations,
	})
}

// Send the direct message
func getOrCreateConversationForUsers(ctx *gin.Context) {
	user1ID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	// Get parameter by name
	user2IDStr := ctx.Param("user2ID")
	user2ID, err := utils.ValidSnowflakeID(user2IDStr)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
	}

	conversation, appErr := conversationStore.GetOrCreateConversation(ctx, user1ID, user2ID)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":      "Conversation find",
		"conversation": conversation,
	})
}

// Get the direct conversation messages if user is part of conversation
func conversationMessagesForUser(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	// Get parameter by name
	conversationID := ctx.Param("conversationID")
	conversationSnowID, err := utils.ValidSnowflakeID(conversationID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
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
		channelID := ctx.Param("channelID")
		channelSnowID, err := utils.ValidSnowflakeID(channelID)
		if err == nil {
			afterCursor = &channelSnowID
		}
	}

	conversation, appErr := conversationStore.GetConversationForUser(ctx, conversationSnowID, userID)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
	}

	messages, appErr := directMessageStore.GetConversationLastMessages(ctx, conversationSnowID, limit, afterCursor)

	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":      "Chat message fetched",
		"conversation": conversation,
		"messages":     messages,
	})
}
