package websocketApp

// Hub (1 goroutine)
//     ├── register chan *Client
//     ├── unregister chan *Client
//     └── broadcast chan []byte

// Each Client (2 goroutines per connection)
//     ├── readPump → reads from conn, sends to broadcast
//     └── writePump ← reads from client.send chan, writes to conn

// Current: user websocket connection good
// TODO:
// What if need to broadcast to specific group say say channel users
// 1. [channel][client]bool
// 2. Before broadcast need to save to db if fail then no broadcast and give the undo message

// Failiure story:
// user switch the channel in between read or write pump ?
// connection switching for channels ? [prev 50 messages]....[websocket messages] user miss some messages
// Do we need to broadcast the previous messages ?

// Improvements Thinking:
// Exponential backoff with jitter
// sync.RWMutex vs sync.Map
// sync.Once - For cleanup
// context for cancellation
// Compression: https://centrifugal.dev/blog/2024/08/19/optimizing-websocket-compression
// https://leapcell.io/blog/building-a-scalable-go-websocket-service-for-thousands-of-concurrent-connections

import (
	"context"
	"discore/internal/modules/core/middlewares"
	"net/http"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
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

// Client represents a single connected WebSocket client.
type Client struct {
	conn   *websocket.Conn
	send   chan []byte // Channel to send messages to the client; // This can panic if you try to close again
	userID snowflake.ID
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

// NewHub creates and returns a new Hub instance.
func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Global hub instance
var (
	globalHub *Hub
	once      sync.Once
)

// Initialize websocket hub
func InitializeHub(ctx context.Context) {
	once.Do(func() {
		// hubCtx, cancel := context.WithCancel(ctx)
		globalHub = newHub()

		// Start the hub's event loop
		go globalHub.run()
	})
}

// run starts the hub's main event loop.
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			logrus.Infof("Client registered: %s (Total: %d)", client.conn.RemoteAddr(), len(h.clients))
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				// close(client.send) // Close the client's send channel
				logrus.Infof("Client unregistered: %s (Total: %d)", client.conn.RemoteAddr(), len(h.clients))
			}
		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// broadcastMessage sends to all clients safely (short lock duration)
func (hub *Hub) broadcastMessage(message []byte) {
	for client := range hub.clients {
		select {
		case client.send <- message:
			// Message sent successfully
		default:
			// If send channel is full, assume client is slow or dead. Unregister and close connection.
			// close(client.send)
			delete(hub.clients, client)
			logrus.WithFields(logrus.Fields{"addr": client.conn.RemoteAddr()}).Warn("Client send channel dead/full, unregistering")
		}
	}
}

// handles WebSocket requests for connections.
func WsHandler(ctx *gin.Context) {
	userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
	if !isOk {
		return
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logrus.WithError(err).Error("Upgrade error")
		return
	}

	conn.SetReadLimit(1024 * 1024) // Add this: reject messages > 1024 KB
	client := &Client{conn: conn, send: make(chan []byte, 256), userID: userID}

	globalHub.register <- client // Register the new client

	go client.writePump()      // Client's write goroutine
	client.readPump(globalHub) // Client's read goroutine (blocks)

	// When readPump exits, unregister the client
	globalHub.unregister <- client
}

// Client related ---

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump(hub *Hub) {
	defer func() {
		// Clean up the client connection when the goroutine exits
		logrus.Infof("Client disconnected: %s", c.conn.RemoteAddr())
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Add deadline
	c.conn.SetPongHandler(func(string) error {               // Add pong handler
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).Error("Read error")
			}
			break
		}

		logrus.Infof("Received: %s", message)
		hub.broadcast <- message
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(time.Second * 10) // Ping interval
	defer func() {
		ticker.Stop()
		c.conn.Close()
		close(c.send)
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Hub closed the channel
				c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logrus.WithError(err).Error("Write error")
				return
			}
		case <-ticker.C:
			// Send ping messages to keep the connection alive
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logrus.WithError(err).Error("Ping error")
				return
			}
		}
	}
}
