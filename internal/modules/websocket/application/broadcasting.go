package websocketApp

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Build the room broadcaster goroutine
func (hub *Hub) BuildRoomBroadcaster(room string) {
	// Race condition could happen as deleting after idle so take lock
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if hub.rooms[room] == nil {
		roomState := hub.NewRoomState(room)
		hub.rooms[room] = roomState
		go hub.roomBroadcaster(room, roomState)
		// logrus.Infof("Started broadcaster for room %s", room)
	}
}

// Room broadcaster; handle the room out buffer requests
func (hub *Hub) roomBroadcaster(room string, roomState *RoomState) {
	ticker := time.NewTicker(150 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case request, ok := <-roomState.outBuffer: // consumer message from room
			if !ok {
				// Closed
				return
			}
			hub.broadcastRequest(request, roomState)

		case <-ticker.C:
			// check if room has any client or not
			roomState.mu.RLock()
			hasClients := len(roomState.clients) > 0
			roomState.mu.RUnlock()

			if !hasClients {
				// logrus.Infof("Stopping broadcaster for room %s (empty)", room)
				hub.removeRoom(room)
				return
			}
		}
	}
}

// Preapare and send message to all clients in room safely
func (hub *Hub) broadcastRequest(broadcastRequest *BroadcastRequest, roomState *RoomState) {
	// No data return immediately
	if broadcastRequest.Data == nil {
		return
	}

	messageBytes, err := json.Marshal(broadcastRequest)
	if err != nil {
		return
	}

	// Optimization: Create prepared message once (compresses once)
	// NOTE: can also use batching to send the messages in batch
	preparedMsg, err := websocket.NewPreparedMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		return
	}

	roomState.mu.RLock()

	var toRemove []*Client // Collect slow clients

	for client := range roomState.clients {
		select {
		case client.send <- preparedMsg:
		default:
			// If send channel is full, assume client is slow or dead. Unregister and close connection.
			// Mark for removal (don't remove while holding RLock)
			toRemove = append(toRemove, client)
			logrus.WithFields(logrus.Fields{"addr": client.conn.RemoteAddr()}).Warn("Client send channel dead/full, unregistering")
		}
	}

	roomState.mu.RUnlock()

	// Remove the clients
	roomState.RemoveClients(toRemove)
}

// Safely deletes a room from hub (called when empty)
func (hub *Hub) removeRoom(room string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	delete(hub.rooms, room)
}

// Safely remove clients from the room
func (room *RoomState) RemoveClients(clients []*Client) {
	// close read, write pump, unregister the client
	room.mu.Lock()
	defer room.mu.Unlock()
	for _, client := range clients {
		delete(room.clients, client)

		if client.conn != nil {
			client.conn.Close() // This makes ReadMessage/WriteMessage return immediately
		}

		if client.done != nil {
			close(client.done)
			client.done = nil
		}
	}

}
