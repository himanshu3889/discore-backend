package websocketApi

import (
	websocketApp "github.com/himanshu3889/discore-backend/internal/modules/websocket/application"
	"github.com/himanshu3889/discore-backend/internal/modules/websocket/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterWebsocketRoutes(rg *gin.RouterGroup) {
	rg.GET("/ws", middlewares.GrpcAuthMiddleware(), websocketApp.WsHandler)
}
