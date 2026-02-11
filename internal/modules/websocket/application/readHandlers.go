package websocketApp

import (
	directmessageService "discore/internal/modules/websocket/services/directMessage"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type EventType string

// Last after . is the action

const (
	// Subscribe event
	EventRoomJoin   EventType = "room.join"
	EventRoomLeave  EventType = "room.leave"
	EventRoomJoined EventType = "room.joined"
	EventRoomTyping EventType = "room.typing"
	// Channel Event
	EventChannelMessageAdd    EventType = "channel-message.add"
	EventChannelMessageUpdate EventType = "channel-message.update"
	EventChannelMessageDelete EventType = "channel-message.delete"
	// Conversation Event
	EventDirectMessageAdd    EventType = "direct-message.add"
	EventDirectMessageUpdate EventType = "direct-message.update"
	EventDirectMessageDelete EventType = "direct-message.delete"
)

type SocketMessage struct {
	Event EventType        `json:"event"`
	Room  string           `json:"room"`
	Data  *json.RawMessage `json:"data"`
}

// Handle the incoming message from the user
func (hub *Hub) HandleIncomingMessage(client *Client, recMessage []byte) {
	var msg SocketMessage
	if err := json.Unmarshal(recMessage, &msg); err != nil {
		// logrus.WithError(err).Warn("Invalid message format")
		return
	}

	// logrus.Infof("Received: %s", recMessage)
	if msg.Room == "" {
		logrus.Warn("Missing room in message")
		return
	}

	switch msg.Event {
	case EventRoomJoin:
		hub.handleRoomJoin(client, msg.Room)
	case EventRoomTyping:
		hub.handleRoomTyping(client, msg.Room)
	case EventChannelMessageAdd:
		hub.handleChannelMessageAdd(client, &msg)
	case EventDirectMessageAdd:
		hub.handleDirectMessageAdd(client, &msg)
	default:
		logrus.Warnf("Unknown event '%s' from user %s", msg.Event, client.userID)
	}
}

// ─────────────────────────────────────────────────────────────────────────────

func _validateClientRoomMessage(client *Client, msg *SocketMessage) error {
	if msg.Data == nil {
		return fmt.Errorf("event %s: nil data", msg.Event)
	}
	if client.room != msg.Room {
		return fmt.Errorf("Invalid room `%s` message", msg.Room)
	}
	return nil
}

func (hub *Hub) handleRoomJoin(client *Client, room string) {
	// Timeout pattern: Allow brief wait for subscribe
	select {
	case hub.subscribe.queue <- &RoomRequest{client: client, name: room}:
		// Sent successfully
	case <-time.After(SubscribeTimeout):
		client.done <- struct{}{}
		return
	}
}

func (hub *Hub) handleRoomTyping(client *Client, room string) {
	hub.mu.RLock()
	roomState, roomExists := hub.rooms[room]
	hub.mu.RUnlock()
	if !roomExists {
		// logrus.Warnf("Room %s not found, message dropped", room)
		return
	}
	roomState.AddTyper(client.userID)
}

func (hub *Hub) handleChannelMessageAdd(client *Client, msg *SocketMessage) {
	if err := _validateClientRoomMessage(client, msg); err != nil {
		return
	}
	if err := hub.producer.Send(
		hub.ctx,
		string(msg.Event),
		msg.Room,
		[]byte(*msg.Data),
		client.userID,
	); err != nil {
		logrus.WithError(err).Error("Kafka publish failed")
	}
}

func (hub *Hub) handleDirectMessageAdd(client *Client, msg *SocketMessage) {
	if err := _validateClientRoomMessage(client, msg); err != nil {
		return
	}

	directMsg, err := directmessageService.SendDirectMessage(msg.Data, client.userID)
	if err != nil {
		return // Don't broadcast on error
	}

	byteMsg, err := json.Marshal(directMsg) // handle errors
	if err != nil {
		return
	}
	var raw json.RawMessage
	err = json.Unmarshal(byteMsg, &raw)
	if err != nil {
		return
	}

	room := msg.Room

	// CRITICAL: Check room exists BEFORE accessing
	hub.mu.RLock()
	roomState, roomExists := hub.rooms[room]
	hub.mu.RUnlock()

	if !roomExists {
		// logrus.Warnf("Room %s not found, message dropped", room)
		return
	}

	// Broadcast to room (fail-fast if room buffer full)
	select {
	case roomState.outBuffer <- &BroadcastRequest{Event: msg.Event, Room: msg.Room, Data: &raw}:
	// Success
	default:
		logrus.Warnf("Room buffer full, dropping message")
	}
}
