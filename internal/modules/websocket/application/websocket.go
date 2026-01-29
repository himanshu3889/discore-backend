package websocketApp

// Hub (1 goroutine)
//     ├── register chan *Client
//     ├── unregister chan *Client
//     └── broadcast chan []byte

// Each Client (2 goroutines per connection)
//     ├── readPump → reads from conn, sends to broadcast
//     └── writePump ← reads from client.send chan, writes to conn

// Current: user websocket connection good
// What if need to broadcast to specific group say say channel users
// 1. [channel][client]bool
// 2. Before broadcast need to save to db if fail then no broadcast and give the undo message
// 3. Think if we need to send something in the server e.g it's server channel delete ?, Server tag, Server announcement

// Failiure story:
// user switch the channel in between read or write pump ?
// connection switching for channels ? [prev 50 messages]....[websocket messages] user miss some messages
// Do we need to broadcast the previous messages ?

// Improvements Thinking:
// context for cancellation
// Compression: https://centrifugal.dev/blog/2024/08/19/optimizing-websocket-compression
// https://leapcell.io/blog/building-a-scalable-go-websocket-service-for-thousands-of-concurrent-connections

import (
	"context"
	"discore/internal/modules/core/middlewares"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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

// handles WebSocket requests for connections.
func WsHandler(ctx *gin.Context) {
	userID, _, _ := middlewares.GetContextUserIDEmail(ctx)
	// if !isOk {
	// 	return
	// }
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logrus.WithError(err).Error("Upgrade error")
		return
	}

	conn.SetReadLimit(1024 * 1024) // Add this: reject messages > 1024 KB
	client := &Client{conn: conn, send: make(chan []byte, 100), userID: userID}

	globalHub.register <- client // Register the new client

	go client.writePump()      // Client's write goroutine
	client.readPump(globalHub) // Client's read goroutine (blocks)

	// When readPump exits, unregister the client
	globalHub.unregister <- client
}

// Constants for connection management
// Server: “Are you still there?” (Ping)
// Client: “Yes, I’m here!” (Pong)
const (
	pingPeriod   = 25 * time.Second
	pingInterval = 10 * time.Second
	missedPings  = 2

	writeWait = 10 * time.Second

	// Time to wait for a pong response
	pongWait = missedPings*pingPeriod + writeWait
)

type UserID = snowflake.ID
type ChannelID = snowflake.ID

// Client represents a single connected WebSocket client.
type Client struct {
	conn    *websocket.Conn
	send    chan []byte // Channel to send messages to the client; // This can panic if you try to close again
	userID  UserID
	viewing string //  the topic
}

// Subscriptions
type View struct {
	client *Client
	view   string // view:unique_id
}

type BroadcastRequest struct {
	Event  EventType        `json:"event"`
	View   string           `json:"view"`
	Data   *json.RawMessage `json:"data"`
	Action string           `json:"action"`
	Client *Client          `json:"-"`
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	// Subscription model, at a time only one view subscribe by client
	view      map[string]map[*Client]bool
	subscribe chan *View // Join view
	// unsubscribe  chan *View // Since one view then Leave view
	totalClients int32 // total client connections

	// Inbound messages from the clients for the view
	broadcast chan *BroadcastRequest

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	mu sync.RWMutex
}

// NewHub creates and returns a new Hub instance.
func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan *BroadcastRequest, 10000),
		register:   make(chan *Client, 10),
		unregister: make(chan *Client),
		view:       make(map[string]map[*Client]bool),
		subscribe:  make(chan *View, 10),
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
func (hub *Hub) run() {
	for {
		select {
		case client := <-hub.register:
			if hub.view[client.viewing] == nil {
				hub.view[client.viewing] = make(map[*Client]bool)
			}
			hub.view[client.viewing][client] = true
			atomic.AddInt32(&hub.totalClients, 1)
			logrus.Infof("Client registered: %s (Total: %d)", client.conn.RemoteAddr(), hub.totalClients)
		case client := <-hub.unregister:
			if _, ok := hub.view[client.viewing][client]; ok {
				delete(hub.view[client.viewing], client)
				// close(client.send) // Close the client's send channel
				atomic.AddInt32(&hub.totalClients, -1)
				logrus.Infof("Client unregistered: %s (Total: %d)", client.conn.RemoteAddr(), hub.totalClients)
			}
		case view := <-hub.subscribe:
			hub.subscribeView(view)

		case broadcastRequest := <-hub.broadcast:
			hub.broadcastRequest(broadcastRequest)
		}
	}
}

// broadcastMessage sends to all clients safely (short lock duration)
func (hub *Hub) broadcastRequest(broadcastRequest *BroadcastRequest) {
	// No data return immediately
	if broadcastRequest.Data == nil {
		return
	}

	hub.mu.RLock()
	clientsMap := hub.view[broadcastRequest.View]

	clients := make([]*Client, 0, len(clientsMap))
	for client := range clientsMap {
		clients = append(clients, client)
	}
	hub.mu.RUnlock()

	messageBytes, err := json.Marshal(broadcastRequest)
	if err != nil {
		return
	}

	for _, client := range clients {
		select {
		case client.send <- messageBytes:
			// Message sent successfully
		default:
			// If send channel is full, assume client is slow or dead. Unregister and close connection.
			// close(client.send)
			delete(hub.view[broadcastRequest.View], client)
			atomic.AddInt32(&hub.totalClients, -1)
			logrus.WithFields(logrus.Fields{"addr": client.conn.RemoteAddr()}).Warn("Client send channel dead/full, unregistering")
		}
	}
}

// Subscribe the particular view
var Allowed_Views_Types = map[string]bool{
	"server": true,
	"direct": true,
}

func (hub *Hub) subscribeView(view *View) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	client := view.client
	newView := view.view

	// Validate the newView type
	newViewParts := strings.SplitN(newView, ":", 2)
	newViewType := ""
	if len(newViewParts) > 0 {
		newViewType = newViewParts[0]
	}
	allowed, ok := Allowed_Views_Types[newViewType]
	validView := (ok && allowed)

	// Remove client from old view (if any)
	oldView := client.viewing
	logrus.Info(validView, " ", newView, " ", oldView)
	if (oldView != "" && oldView != newView) || !validView {
		if clients, ok := hub.view[oldView]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(hub.view, oldView)
			}
		}
		if !validView {
			return
		}
	}

	// init inner map if not present
	if hub.view[newView] == nil {
		hub.view[newView] = make(map[*Client]bool)
	}

	// add/update client
	hub.view[newView][view.client] = true
	client.viewing = newView
}

// Client related ---

// readPump reads messages from the WebSocket connection.
func (client *Client) readPump(hub *Hub) {
	defer func() {
		// Clean up the client connection when the goroutine exits
		logrus.Infof("Client disconnected: %s", client.conn.RemoteAddr())
		client.conn.Close()
	}()

	client.conn.SetReadDeadline(time.Now().Add(pongWait)) // Add deadline
	client.conn.SetPongHandler(func(string) error {       // Add pong handler
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, recMessage, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).Error("Read error")
			}
			break
		}

		var msg InboundMessage
		if err := json.Unmarshal(recMessage, &msg); err != nil {
			logrus.WithError(err).Warn("Invalid message format")
			continue
		}

		// Verifying the message based on the "event"
		// based on the event the things will broadcast

		logrus.Infof("Received: %s", recMessage)
		if msg.View == "" {
			logrus.Warn("Missing view in message")
			continue
		}

		// First subscribe
		hub.subscribe <- &View{client: client, view: msg.View}

		// Broadcast if we have data
		if msg.Data == nil {
			continue
		}

		// based on event we will do some more things before broadcast
		outboundMsg := hub.handleWebsocketMessage(client.userID, &msg)

		// If data is null then no broadcasting
		if outboundMsg.Data == nil {
			continue
		}

		hub.broadcast <- &BroadcastRequest{
			Event:  outboundMsg.Event,
			View:   outboundMsg.View,
			Data:   outboundMsg.Data,
			Action: msg.Action,
			Client: client,
		}
	}
}

// writePump writes messages to the WebSocket connection.
func (client *Client) writePump() {
	ticker := time.NewTicker(pingInterval) // Ping interval
	defer func() {
		ticker.Stop()
		client.conn.Close()
		close(client.send)
	}()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				// Hub closed the channel
				client.conn.SetWriteDeadline(time.Now().Add(writeWait))
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logrus.WithError(err).Error("Write error")
				return
			}
		case <-ticker.C:
			// Send ping messages to keep the connection alive
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logrus.WithError(err).Error("Ping error")
				return
			}
		}
	}
}
