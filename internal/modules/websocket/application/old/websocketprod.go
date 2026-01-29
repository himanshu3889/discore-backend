package websocketApp

// import (
// 	"context"
// 	"discore/internal/base/utils"
// 	"discore/internal/modules/core/middlewares"
// 	"net/http"
// 	"sync"
// 	"time"

// 	"github.com/bwmarrin/snowflake"
// 	"github.com/gin-gonic/gin"
// 	"github.com/gorilla/websocket"
// 	"github.com/sirupsen/logrus"
// )

// // Configure the upgrader
// var upgrader = websocket.Upgrader{
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true
// 	},
// }

// type UserID = snowflake.ID

// // Client represents a single WebSocket connection
// type Client struct {
// 	hub       *Hub
// 	conn      *websocket.Conn
// 	send      chan []byte // Buffered channel for outbound messages
// 	userID    UserID      // The user this connection belongs to
// 	deviceID  string      // Unique device/session identifier
// 	closeOnce sync.Once
// }

// // Hub maintains the set of active clients and broadcasts messages
// type Hub struct {
// 	clients    map[UserID]*Client // One client per user (easily extendable to multiple)
// 	register   chan *Client       // Add new client
// 	unregister chan *Client       // Remove client
// 	broadcast  chan []byte        // Messages to broadcast
// 	mu         sync.RWMutex       // Protects clients map for snapshots
// 	ctx        context.Context    // For graceful shutdown
// 	cancel     context.CancelFunc // Cancel function
// }

// // Global hub instance
// var (
// 	globalHub *Hub
// 	once      sync.Once
// )

// // Initialize websocket hub
// func InitializeHub(ctx context.Context) {
// 	once.Do(func() {
// 		hubCtx, cancel := context.WithCancel(ctx)
// 		globalHub = &Hub{
// 			clients:    make(map[UserID]*Client),
// 			register:   make(chan *Client),
// 			unregister: make(chan *Client),
// 			broadcast:  make(chan []byte, 256),
// 			ctx:        hubCtx,
// 			cancel:     cancel,
// 		}

// 		// Start the hub's event loop
// 		go globalHub.Run()
// 	})
// }

// // Shutdown gracefully stops the hub
// func Shutdown() {
// 	if globalHub != nil {
// 		globalHub.cancel()
// 	}
// }

// // Run is the Hub's main event loop (single goroutine, thread-safe)
// func (hub *Hub) Run() {
// 	for {
// 		select {
// 		case <-hub.ctx.Done():
// 			logrus.Info("Hub shutting down")
// 			hub.mu.Lock()
// 			for _, client := range hub.clients {
// 				client.shutdown()
// 			}
// 			hub.clients = make(map[UserID]*Client)
// 			hub.mu.Unlock()
// 			return

// 		case client := <-hub.register:
// 			hub.mu.Lock()
// 			// If user reconnects, close old connection
// 			if oldClient, exists := hub.clients[client.userID]; exists {
// 				logrus.WithField("userID", client.userID).Info("Closing old connection for user")
// 				oldClient.shutdown()
// 			}
// 			hub.clients[client.userID] = client
// 			hub.mu.Unlock()
// 			logrus.WithFields(logrus.Fields{
// 				"userID":   client.userID,
// 				"deviceID": client.deviceID,
// 				"total":    len(hub.clients),
// 			}).Info("Client registered")

// 		case client := <-hub.unregister:
// 			hub.mu.Lock()
// 			// Verify the client in the map is the one unregistering
// 			if current, exists := hub.clients[client.userID]; exists && current == client {
// 				delete(hub.clients, client.userID)
// 				hub.mu.Unlock() // Unlock BEFORE shutdown
// 				client.shutdown()
// 				logrus.WithFields(logrus.Fields{
// 					"userID":   client.userID,
// 					"deviceID": client.deviceID,
// 					"total":    len(hub.clients),
// 				}).Info("Client unregistered")
// 			} else {
// 				hub.mu.Unlock()
// 				logrus.WithField("userID", client.userID).Debug("Client already replaced")
// 			}

// 		case message := <-hub.broadcast:
// 			hub.broadcastMessage(message)
// 		}
// 	}
// }

// // shutdown closes the client safely (idempotent)
// func (c *Client) shutdown() {
// 	c.closeOnce.Do(func() {
// 		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
// 		c.conn.WriteMessage(websocket.CloseMessage,
// 			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
// 		close(c.send)
// 		c.conn.Close()
// 	})
// }

// // broadcastMessage sends to all clients safely (short lock duration)
// func (hub *Hub) broadcastMessage(message []byte) {
// 	// Lock only for iteration (microseconds)
// 	hub.mu.RLock()
// 	for _, client := range hub.clients {
// 		select {
// 		case client.send <- message: // Non-blocking send
// 		default:
// 			// Buffer full = dead client
// 			// TODO: too many fulls so too many goroutines
// 			go func(c *Client) { hub.unregister <- c }(client)
// 		}
// 	}
// 	hub.mu.RUnlock()
// }

// // WebSocket handler for Gin
// func WsHandler(ctx *gin.Context) {
// 	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
// 	if !isOk {
// 		return
// 	}

// 	// TODO: Generate device/session ID (use header or generate unique)
// 	deviceID := "web:" + utils.GenerateSnowflakeID().String()

// 	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
// 	if err != nil {
// 		logrus.WithFields(logrus.Fields{
// 			"event": "ws_error",
// 		}).WithError(err).Error("Failed to upgrade WebSocket connection")
// 		return
// 	}

// 	client := &Client{
// 		hub:      globalHub,
// 		conn:     conn,
// 		send:     make(chan []byte, 256), // Buffered to prevent blocking
// 		userID:   userID,
// 		deviceID: deviceID,
// 	}

// 	ready := make(chan struct{})
// 	go client.writePump(ready)
// 	go client.readPump()
// 	<-ready // Wait for writePump to be ready
// 	// Register client with hub (thread-safe via channel)
// 	globalHub.register <- client

// 	logrus.WithFields(logrus.Fields{
// 		"userID":   userID,
// 		"deviceID": deviceID,
// 		"remote":   conn.RemoteAddr(),
// 	}).Info("WebSocket connection established")
// }

// // readPump pumps messages from websocket connection to the hub
// func (c *Client) readPump() {
// 	defer func() {
// 		c.hub.unregister <- c
// 		c.conn.Close()
// 	}()

// 	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
// 	c.conn.SetReadDeadline(time.Now().Add(pongWait))
// 	c.conn.SetPongHandler(func(string) error {
// 		c.conn.SetReadDeadline(time.Now().Add(pongWait))
// 		return nil
// 	})

// 	for {
// 		_, message, err := c.conn.ReadMessage()
// 		if err != nil {
// 			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
// 				logrus.WithError(err).WithField("userID", c.userID).Warn("WebSocket read error")
// 			}
// 			break
// 		}

// 		logrus.WithFields(logrus.Fields{
// 			"userID":   c.userID,
// 			"deviceID": c.deviceID,
// 			"message":  string(message),
// 		}).Debug("Received message from client")

// 		// Broadcast to all clients
// 		globalHub.broadcast <- message
// 	}
// }

// // writePump pumps messages from the hub to the websocket connection
// func (c *Client) writePump(ready chan struct{}) {
// 	ticker := time.NewTicker(pingPeriod)
// 	defer func() {
// 		ticker.Stop()
// 		c.conn.Close()
// 	}()

// 	for {
// 		select {
// 		case message, ok := <-c.send:
// 			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
// 			if !ok {
// 				c.conn.SetWriteDeadline(time.Now().Add(writeWait))
// 				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
// 				return
// 			}

// 			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
// 				logrus.WithError(err).WithField("userID", c.userID).Warn("WebSocket write error")
// 				return
// 			}

// 		case <-ticker.C:
// 			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
// 			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
// 				logrus.WithError(err).WithField("userID", c.userID).Warn("WebSocket ping error")
// 				return
// 			}
// 		}
// 	}
// }

// // Broadcast provides a safe way to send messages from outside
// func Broadcast(message []byte) {
// 	select {
// 	case globalHub.broadcast <- message:
// 	case <-time.After(100 * time.Millisecond): // Timeout after 100ms
// 		logrus.Warn("Broadcast dropped: hub saturated")
// 	}
// }

// // Constants for connection management
// // Server: “Are you still there?” (Ping)
// // Client: “Yes, I’m here!” (Pong)
// const (
// 	pingPeriod  = 25 * time.Second
// 	missedPings = 2

// 	writeWait = 10 * time.Second

// 	// Time to wait for a pong response
// 	pongWait = missedPings*pingPeriod + writeWait
// )
