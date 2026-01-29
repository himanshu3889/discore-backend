package websocketApi

import (
	"discore/internal/modules/core/middlewares"
	websocketApp "discore/internal/modules/websocket/application"

	"github.com/gin-gonic/gin"
)

func RegisterWebsocketRoutes(rg *gin.RouterGroup) {
	rg.GET("/ws", middlewares.JwtAuthMiddleware(), websocketApp.WsHandler)
}
