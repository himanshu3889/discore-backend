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
		roomState := hub.GetOrCreateRoom(room)
		go hub.roomBroadcaster(room, roomState)
		// logrus.Infof("Started broadcaster for room %s", room)
	}
}

// Room broadcaster; handle the room out buffer requests
func (hub *Hub) roomBroadcaster(room string, roomState *RoomState) {
	// Configuration for the batching window
	const (
		batchTimeout = 50 * time.Millisecond // Wait max 50ms
		maxBatchSize = 50                    // wait for 50 messages
	)

	removeRoomTicker := time.NewTicker(150 * time.Second)
	defer removeRoomTicker.Stop()

	flushTicker := time.NewTicker(batchTimeout)
	defer flushTicker.Stop()

	batch := make([]*BroadcastRequest, 0, maxBatchSize)

	// Helper closure to trigger the send
	flushBatchRequests := func() {
		if len(batch) == 0 {
			return
		}

		hub.broadcastBatchRequest(batch, roomState)

		flushTicker.Reset(batchTimeout) // reset ticker as we flushed
		// Clear the batch but keep the underlying array capacity, This prevents Garbage Collection churn.
		batch = batch[:0]
	}

	for {
		select {
		case request, ok := <-roomState.outBuffer: // consumer message from room
			if !ok {
				// Closed
				return
			}

			// hub.broadcastRequest(request, roomState) // previously single message flushed instantly

			// Now batching :

			batch = append(batch, request)

			// INSTANT DRAIN ("The Greedy Loop")
			// If there are more messages waiting in the buffer RIGHT NOW,
			// grab them immediately without spinning the outer loop.
			queuedCount := len(roomState.outBuffer)
			for i := 0; i < queuedCount; i++ {
				// Stop if we hit the batch limit
				if len(batch) >= maxBatchSize {
					break
				}
				// We know data is there, so this receive will happen instantly
				batch = append(batch, <-roomState.outBuffer)
			}

			if len(batch) >= maxBatchSize {
				flushBatchRequests()
			}

		case <-flushTicker.C:
			flushBatchRequests()

		case <-removeRoomTicker.C:
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

// [Deprecated] Preapare and send message to all clients in room safely
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

	// [METRIC] Start the timer before entering the critical section
	broadcastStart := time.Now()

	roomState.mu.RLock()

	var toRemove []*Client // Collect slow clients

	for client := range roomState.clients {
		select {
		case client.send <- preparedMsg:
		default:
			// If send channel is full, assume client is slow or dead. Unregister and close connection.
			// Mark for removal (don't remove while holding RLock)
			toRemove = append(toRemove, client)
			// logrus.WithFields(logrus.Fields{"addr": client.conn.RemoteAddr()}).Warn("Client send channel dead/full, unregistering")
		}
	}

	roomState.mu.RUnlock()

	// [METRIC] Record immediately after releasing lock.
	hub.MetricRecordBroadcast(
		broadcastRequest.Event,
		broadcastStart,
		broadcastRequest.PipelineStart,
	)

	// Remove the clients
	roomState.RemoveClients(toRemove)
}

// compresses the batch ONCE and sends it to all clients
func (hub *Hub) broadcastBatchRequest(messages []*BroadcastRequest, roomState *RoomState) {
	// Marshal the entire ARRAY of messages
	// Output JSON: [{"event":"msg", "data":"hi"}, {"event":"msg", "data":"hello"}]
	batchBytes, err := json.Marshal(messages)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal batch")
		return
	}

	// Create ONE PreparedMessage (Compresses ONCE for all clients)
	preparedMsg, err := websocket.NewPreparedMessage(websocket.TextMessage, batchBytes)
	if err != nil {
		logrus.WithError(err).Error("Failed to prepare message")
		return
	}

	// [METRIC] Start the timer before entering the critical section
	broadcastStart := time.Now()

	// Lock ONCE for the entire batch
	roomState.mu.RLock()

	var toRemove []*Client // Collect slow clients

	for client := range roomState.clients {
		select {
		case client.send <- preparedMsg:
			// Message queued successfully
		default:
			// If send channel is full, assume client is slow or dead. Unregister and close connection.
			// Mark for removal (don't remove while holding RLock)
			toRemove = append(toRemove, client)
			// logrus.WithFields(logrus.Fields{"addr": client.conn.RemoteAddr()}).Warn("Client send channel dead/full, unregistering")
		}
	}

	roomState.mu.RUnlock()

	// Remove the clients
	roomState.RemoveClients(toRemove)

	for _, req := range messages {
		hub.MetricRecordBroadcast(
			req.Event,
			broadcastStart,
			req.PipelineStart,
		)
	}

}

// Safely deletes a room from hub (called when empty)
func (hub *Hub) removeRoom(room string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if _, exists := hub.rooms[room]; exists {
		delete(hub.rooms, room)
		hub.MetricTrackRoom(false) // Decrease Gauge
	}
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
