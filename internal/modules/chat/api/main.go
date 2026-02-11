package chatApi

import (
	"discore/internal/modules/chat/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(rg *gin.RouterGroup) {
	chatGrp := rg.Group("/chat/api", middlewares.AuthMiddleware())

	registerChannelMessageRoutes(chatGrp)
	registerConversationRoutes(chatGrp)
}
