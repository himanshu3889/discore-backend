package coreApi

import (
	"github.com/himanshu3889/discore-backend/base/middlewares"
	"github.com/himanshu3889/discore-backend/base/utils"

	"net/http"

	serverStore "github.com/himanshu3889/discore-backend/base/store/server"

	"github.com/gin-gonic/gin"
)

func registerMemberRoutes(r *gin.RouterGroup) {
	memberGroup := r.Group("/members")
	memberRoutes(memberGroup)
}

func memberRoutes(rg *gin.RouterGroup) {
	rg.GET("/user/server/:serverID", GetUserServerMember)
	rg.GET("/user/profile/server/:serverID", GetUserServerMemberProfile)
	// Get members in the server; if user is in the server; paginated
}

// Get the member in the user joined server
func GetUserServerMember(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	// Get parameter by name
	serverID := ctx.Param("serverID")
	serverSnowID, err := utils.ValidSnowflakeID(serverID)

	// Validate it's a valid ID
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	member, appErr := serverStore.GetUserServerMemember(ctx, userID, serverSnowID)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"member": member, "message": "Channel found"})
}

// Get the single user with membership in the user joined server
func GetUserServerMemberProfile(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	// Get parameter by name
	serverID := ctx.Param("serverID")
	serverSnowID, err := utils.ValidSnowflakeID(serverID)

	// Validate it's a valid ID
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// Check first if user is the member in the server or not
	member, appErr := serverStore.GetUserServerMemember(ctx, userID, serverSnowID)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"member": member, "message": "Channel found"})
}
