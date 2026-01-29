package websocketApp

import (
	chatModel "discore/internal/modules/chat/models"
	channelMessage "discore/internal/modules/chat/services/websocket"
	"encoding/json"

	"github.com/bwmarrin/snowflake"
)

type EventType string

const (
	EventChannelMessage EventType = "channel-message"
	EventServerJoin     EventType = "server-join"
	EventChannelLeave   EventType = "channel-leave"
)

type InboundMessage struct {
	Event  EventType        `json:"event"`
	View   string           `json:"view"`
	Action string           `json:"action"`
	Data   *json.RawMessage `json:"data"`
}

func (hub *Hub) handleWebsocketMessage(userID snowflake.ID, msg *InboundMessage) *BroadcastRequest {
	var data *json.RawMessage
	switch msg.Event {
	case EventChannelMessage:
		channelMsg, err := addMessageToChannel(userID, msg)
		if err != nil {
			break
		}
		// Marshal the struct to JSON bytes
		msgBytes, err := json.Marshal(channelMsg)
		if err != nil {
			break
		}

		// Convert to RawMessage and take address
		raw := json.RawMessage(msgBytes)
		data = &raw
	}
	return &BroadcastRequest{Event: msg.Event, View: msg.View, Data: data}
}

func addMessageToChannel(userID snowflake.ID, msg *InboundMessage) (*chatModel.ChannelMessage, error) {
	return channelMessage.HandleChannelRawMessage(userID, *msg.Data)
}
