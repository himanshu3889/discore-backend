package chatApi

import (
	"github.com/himanshu3889/discore-backend/base/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(rg *gin.RouterGroup) {
	chatGrp := rg.Group("/chat/api", middlewares.PassportAuthMiddleware())

	registerChannelMessageRoutes(chatGrp)
	registerConversationRoutes(chatGrp)
}
