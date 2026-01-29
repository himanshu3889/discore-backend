package coreApi

import (
	"discore/internal/base/utils"
	"discore/internal/modules/core/middlewares"
	accountStore "discore/internal/modules/core/store/account"
	memberStore "discore/internal/modules/core/store/member"
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerMemberRoutes(r *gin.RouterGroup) {
	memberGroup := r.Group("/members", middlewares.JwtAuthMiddleware())
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

	member, err := memberStore.GetUserServerMemember(ctx, userID, serverSnowID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"member": member, "message": "Channel found"})
}

// Get the user and member in the user joined server
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
	member, err := memberStore.GetUserServerMemember(ctx, userID, serverSnowID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Get the user
	user, err := accountStore.GetUserByID(ctx, userID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"member": member, "user": user, "message": "Channel found"})
}
