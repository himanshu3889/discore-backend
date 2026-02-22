package coreApi

import (
	"net/http"

	channelCacheStore "github.com/himanshu3889/discore-backend/base/cacheStore/channel"
	serverCacheStore "github.com/himanshu3889/discore-backend/base/cacheStore/server"
	"github.com/himanshu3889/discore-backend/base/middlewares"
	"github.com/himanshu3889/discore-backend/base/models"
	memberStore "github.com/himanshu3889/discore-backend/base/store/member"
	serverStore "github.com/himanshu3889/discore-backend/base/store/server"
	"github.com/himanshu3889/discore-backend/base/utils"

	"github.com/gin-gonic/gin"
)

func registerServerRoutes(r *gin.RouterGroup) {
	serverGroup := r.Group("/servers")
	serverRoutes(serverGroup)

}

func serverRoutes(rg *gin.RouterGroup) {
	rg.POST("", CreateServer)
	rg.PATCH("/:serverID", EditServer)
	rg.GET("/user/first-joined", UserFirstJoinedServer)
	rg.GET("/:serverID/user", UserServer)
	rg.GET("/user/all-joined", UserAllJoinedServers)
	rg.GET("/:serverID/user/channels", UserServerChannels)
	rg.POST("/:serverID/invite-code", CreateServerInvite)
	rg.POST("/accept-invite/:inviteCode", AcceptServerInvite)
	rg.GET("/:serverID/members", GetServerMembers)
}

// User first joined server
func UserFirstJoinedServer(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	firstServer, err := serverStore.UserFirstJoinedServer(ctx, userID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if firstServer == nil {
		utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"server": nil, "message": "User has not joined any servers"})
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"server": firstServer, "message": "Server found"})
}

// Get User server details
func UserServer(ctx *gin.Context) {
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

	userServer, member, err := serverStore.GetServerMembershipForUser(ctx, serverSnowID, userID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if userServer == nil {
		utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"server": nil, "member": nil, "message": "User has not joined any servers"})
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"server": userServer, "member": member, "message": "Server found"})
}

// Get User server channels
func UserServerChannels(ctx *gin.Context) {
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

	server, member, err := serverStore.GetServerMembershipForUser(ctx, serverSnowID, userID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if server == nil {
		// FIXME: return error as invalid access
		utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"server": nil, "member": nil, "message": "User has not member of the server"})
		return
	}

	serverChannels, err := serverCacheStore.GetServerChannels(ctx, serverSnowID)

	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"server": server, "member": member, "channels": serverChannels, "message": "Server found"})
}

// Get User server members; user should be member of the server
func GetServerMembers(ctx *gin.Context) {
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

	userServer, _, err := serverStore.GetServerMembershipForUser(ctx, serverSnowID, userID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if userServer == nil {
		utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"server": nil, "message": "User has not member of the server"})
		return
	}

	members, err := serverCacheStore.GetServerMembers(ctx, serverSnowID, 50, 0)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"server": userServer, "members": members})
}

// User all joined server
func UserAllJoinedServers(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	joinedServers, err := serverStore.UserJoinedServers(ctx, userID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if joinedServers == nil {
		utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"server": nil, "message": "User has not joined any servers"})
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"servers": joinedServers, "message": "Server found"})
}

// Create a new server for the user with the general channel
func CreateServer(ctx *gin.Context) {

	var incomingServer *models.Server
	if err := ctx.ShouldBindJSON(&incomingServer); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	// create server
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}
	incomingServer.OwnerID = userID
	err := serverCacheStore.CreateServer(ctx, incomingServer)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	// create a general channel for it
	var createdChannel = &models.Channel{
		Name:      "General",
		Type:      models.ChannelTypeText,
		CreatorID: incomingServer.OwnerID,
		ServerID:  incomingServer.ID,
	}
	err = channelCacheStore.CreateChannel(ctx, createdChannel)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// User join the server as a Member (Admin)
	var createdMember = &models.Member{
		Role:     models.MemberRoleADMIN,
		UserID:   incomingServer.OwnerID,
		ServerID: incomingServer.ID,
	}
	err = memberStore.CreateMember(ctx, createdMember)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusCreated, gin.H{
		"message":  "Server created successfully",
		"server":   incomingServer,
		"channels": []*models.Channel{createdChannel},
	})

}

// Edit a server for the user with the general channel
func EditServer(ctx *gin.Context) {

	var incomingServer *models.Server
	if err := ctx.ShouldBindJSON(&incomingServer); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	// Get parameter by name
	serverID := ctx.Param("serverID")
	serverSnowID, err := utils.ValidSnowflakeID(serverID)

	incomingServer.ID = serverSnowID

	// Validate it's a valid ID
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// Check if user is the owner of the server or not
	hasOwn, err := serverStore.HasUserOwnServer(ctx, userID, serverSnowID)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	if !hasOwn {
		utils.RespondWithError(ctx, http.StatusForbidden, "Request forbidden")
		return
	}

	err = serverCacheStore.UpdateServerNameImage(ctx, incomingServer)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusCreated, gin.H{
		"message": "Server created successfully",
		"server":  incomingServer,
	})

}

// Create server invite for the user
func CreateServerInvite(ctx *gin.Context) {
	// create server invite
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	var incomingServerInvite *models.ServerInvite
	if err := ctx.ShouldBindJSON(&incomingServerInvite); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	serverID := ctx.Param("serverID")
	serverSnowID, err := utils.ValidSnowflakeID(serverID)

	incomingServerInvite.ServerID = serverSnowID
	incomingServerInvite.CreatedBy = userID
	err = serverCacheStore.CreateServerInvite(ctx, incomingServerInvite)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusCreated, gin.H{
		"message": "Server invite created successfully",
		"invite":  incomingServerInvite,
	})
}

// Create server invite for the user
func AcceptServerInvite(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid token")
		return
	}

	inviteCode := ctx.Param("inviteCode")

	incomingServerInvite := &models.ServerInvite{
		Code: inviteCode,
	}

	serverInvite, err := serverCacheStore.AcceptServerInviteAndCreateMember(ctx, userID, incomingServerInvite.Code)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, "Unable to accept server invite")
		return
	}
	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":     "Server invite accepted",
		"invite_code": incomingServerInvite.Code,
		"server_id":   serverInvite.ServerID,
	})
}
