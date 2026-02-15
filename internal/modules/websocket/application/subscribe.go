package websocketApp

import (
	"context"
	"discore/internal/base/utils"
	serverCacheStore "discore/internal/modules/websocket/cacheStore/server"
	directmessageStore "discore/internal/modules/websocket/store/directMessage"
	"encoding/json"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type VIEW_TYPE string

const (
	SERVER_ROOM VIEW_TYPE = "server"
	DIRECT_ROOM VIEW_TYPE = "direct"
)

// Subscribe the particular room
var Allowed_Room_Types = map[VIEW_TYPE]bool{
	SERVER_ROOM: true,
	DIRECT_ROOM: true,
}

var SubscribeTimeout = 3 * time.Second

// Subscribe the room
func (hub *Hub) SubscribeRoom(roomReq *RoomRequest) {
	client := roomReq.client
	newRoomName := roomReq.name

	// --- Validation (outside lock) ---
	newRoomParts := strings.SplitN(newRoomName, ":", 2)
	if len(newRoomParts) != 2 {
		logrus.Warnf("Invalid room name format: %s", newRoomName)
		return
	}

	newRoomType := VIEW_TYPE(newRoomParts[0])
	newRoomTypeID, err := utils.ValidSnowflakeID(newRoomParts[1])
	if err != nil {
		logrus.Warnf("Invalid room")
		return
	}

	allowed, ok := Allowed_Room_Types[newRoomType]
	if !ok || !allowed {
		logrus.Warnf("Room type not allowed: %s", newRoomType)
		return
	}

	oldRoomName := client.room
	if oldRoomName == newRoomName {
		return // Already in target room
	}

	// --- Get room states (hub lock only for map ops) ---
	hub.mu.Lock()
	var oldRoom *RoomState
	if oldRoomName != "" {
		oldRoom = hub.rooms[oldRoomName]
	}

	newRoom := hub.rooms[newRoomName]
	if newRoom == nil {
		hub.GetOrCreateRoom(newRoomName)
	}
	hub.mu.Unlock()

	// --- Switch rooms (per-room locks only) ---
	// Remove old room
	if oldRoom != nil {
		oldRoom.RemoveClients([]*Client{client})
	}

	// Lock new room and add client

	var canJoin bool
	switch newRoomType {
	case SERVER_ROOM:
		canJoin = newRoom.canClientInServerRoom(hub.ctx, client.userID, newRoomTypeID)
	case DIRECT_ROOM:
		canJoin = newRoom.canClientInDirectRoom(hub.ctx, client.userID, newRoomTypeID)
	}

	if !canJoin {
		return
	}

	newRoom.addClient(client, newRoomName)
	client.sendSubscribeConfirmation()

}

// Check client user join the server room
func (room *RoomState) canClientInServerRoom(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) bool {
	canJoin, _ := serverCacheStore.HasUserServerMember(ctx, userID, serverID)
	return canJoin
}

// Can client user join the dm room
func (room *RoomState) canClientInDirectRoom(ctx context.Context, userID snowflake.ID, conversationID snowflake.ID) bool {
	canJoin, _ := directmessageStore.HasValidConversationForUser(ctx, conversationID, userID)
	return canJoin
}

// Add client in the room safely
func (room *RoomState) addClient(client *Client, roomName string) {
	room.mu.Lock()
	defer room.mu.Unlock()
	room.clients[client] = true
	client.room = roomName
}

// Subscribe confirmation once user join the room
func (client *Client) sendSubscribeConfirmation() {
	data := json.RawMessage(`{"success": true, "message": "Successfully joined room"}`)
	var broadcastRequest = &BroadcastRequest{
		Event: EventRoomJoined,
		Room:  client.room,
		Data:  &data,
	}
	messageBytes, err := json.Marshal(broadcastRequest)
	if err != nil {
		return
	}
	preparedMsg, err := websocket.NewPreparedMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		return
	}
	client.send <- preparedMsg
}

// Initialize subscriber workers
func (hub *Hub) InitSubscribeWorkers() {
	for i := 0; i < hub.subscribe.workers; i++ {
		hub.subscribe.wg.Add(1)
		go hub.subscribeWorker(hub.ctx)
	}
}

// worker to subscribe room
func (hub *Hub) subscribeWorker(ctx context.Context) {
	defer hub.subscribe.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case roomRequest, ok := <-hub.subscribe.queue:
			if !ok {
				// Closed
				return
			}

			// [METRIC] : Job picked up
			hub.MetricSubscribeQueueDepth(true)

			// Process job
			hub.BuildRoomBroadcaster(roomRequest.name)
			hub.SubscribeRoom(roomRequest)

		}
	}
}
