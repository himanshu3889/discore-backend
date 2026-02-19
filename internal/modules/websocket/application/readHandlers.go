package websocketApp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/himanshu3889/discore-backend/base/models"
	"github.com/himanshu3889/discore-backend/base/utils"
	directmessageService "github.com/himanshu3889/discore-backend/internal/modules/websocket/services/directMessage"

	"github.com/segmentio/kafka-go"
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
	Event         EventType        `json:"event"`
	Room          string           `json:"room"`
	Data          *json.RawMessage `json:"data"`
	PipelineStart time.Time        `json:"-"`
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

	msg.PipelineStart = time.Now()

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
		// [METRIC] Success! Item is now in the queue.
		hub.MetricSubscribeQueueDepth(false)
	case <-time.After(SubscribeTimeout):
		client.done <- struct{}{}
		// [METRIC]: Track dropped connections due to full queue
		hub.MetricSubscribeQueueDrop()
		return
	}
}

func (hub *Hub) handleRoomTyping(client *Client, room string) {
	hub.mu.RLock()
	roomState, roomExists := hub.rooms[room]
	hub.mu.RUnlock()
	if !roomExists {
		logrus.Warnf("Room %s not found for typing, message dropped", room)
		return
	}

	// [METRIC]
	hub.MetricRoomTyping()
	roomState.AddTyper(client.userID)
}

func (hub *Hub) handleChannelMessageAdd(client *Client, msg *SocketMessage) {
	if err := _validateClientRoomMessage(client, msg); err != nil {
		return
	}

	broadcastTopic := "broadcast." + msg.Event // FIXME: genererate it using func ?

	msgID := utils.GenerateSnowflakeID()

	ingestHeader := kafka.Header{
		Key:   "ingest_time",
		Value: []byte(fmt.Sprintf("%d", msg.PipelineStart.UnixMilli())),
	}
	// Preserve the trace_id
	traceHeader := kafka.Header{
		Key:   "trace_id",
		Value: []byte(msgID.String()),
	}

	// Push in kafka to write to db
	if err := hub.producer.Send(hub.ctx,
		string(msg.Event),
		msg.Room,
		[]byte(*msg.Data),
		client.userID,
		traceHeader,
		ingestHeader,
	); err != nil {
		logrus.WithError(err).Error("Kafka publish channel message failed")
		return
	}

	var incomingMessage models.ChannelMessage
	incomingMessage.ID = msgID
	incomingMessage.UserID = client.userID

	if err := json.Unmarshal(*msg.Data, &incomingMessage); err != nil {
		logrus.WithError(err).Warn("Invalid message format")
		return
	}

	createdMessageBytes, err := json.Marshal(incomingMessage)
	if err != nil {
		return
	}

	// Push in kafka for broadcast
	if err := hub.producer.Send(hub.ctx,
		string(broadcastTopic),
		msg.Room,
		createdMessageBytes,
		client.userID,
		traceHeader,
		ingestHeader,
	); err != nil {
		logrus.WithError(err).Error("Failed to forward to broadcast topic")
		return
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
	case roomState.outBuffer <- &BroadcastRequest{Event: msg.Event, Room: msg.Room, Data: &raw, PipelineStart: msg.PipelineStart}:
	// Success
	default:
		logrus.Warnf("Room buffer full, dropping message")
	}
}
