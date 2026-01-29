package chatApi

import "github.com/gin-gonic/gin"

func RegisterChatRoutes(rg *gin.RouterGroup) {
	chatGrp := rg.Group("/chat/api")

	registerMessageRoutes(chatGrp)
}
