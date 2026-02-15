package websocketApp

import (
	"discore/internal/modules/core/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Upgrader upgrades HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for simplicity in this example.
		// In production, restrict this to your domain.
		return true
	},
}

// handles WebSocket requests for connections.
func WsHandler(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		logrus.Error("Invalid userID")
		return
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logrus.WithError(err).Error("Upgrade error")
		return
	}

	conn.SetReadLimit(1024 * 1024) // Add this: reject messages > 1024 KB

	client := &Client{conn: conn, send: make(chan *websocket.PreparedMessage, clientBufferSize), userID: userID, done: make(chan struct{})}

	globalHub.register <- client // Register the new client

	go client.WritePump()      // Client's write goroutine
	client.ReadPump(globalHub) // Client's read goroutine (blocks)

	// Blocked by client readpump, When readPump exits, unregister the client
	globalHub.unregister <- client
}
