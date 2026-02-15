package websocketApp

import (
	"discore/configs"
	baseKafka "discore/internal/base/infrastructure/kafka"
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Make handler for channel broadcasting
func makeChannelBroadcastHandler(hub *Hub) func(*kafka.Message) error {
	return func(msg *kafka.Message) error {
		// logrus.Infof("RAW JSON in broadcasting: %s\n", string(msg.Value))

		rawData := &json.RawMessage{}
		err := json.Unmarshal(msg.Value, rawData)
		if err != nil {
			return nil
		}

		kafkaMetadata := baseKafka.ParseKafkaMessageHeaders(msg)

		msgTopic := "channel-message.add"
		room := string(msg.Key)

		// CRITICAL: Check room exists BEFORE accessing
		hub.mu.RLock()
		roomState, roomExists := hub.rooms[room]
		hub.mu.RUnlock()

		if !roomExists {
			// logrus.Warnf("Room %s not found, message dropped", room)
			return nil
		}

		var socketMessage = &BroadcastRequest{
			Event:         EventType(msgTopic),
			Room:          room,
			Data:          rawData,
			PipelineStart: kafkaMetadata.IngestTime,
		}

		select {
		case roomState.outBuffer <- socketMessage: // send message to room
			return nil
		case <-time.After(5 * time.Second):
			// Optional: timeout if workers stuck too long
			logrus.Warn("Workers overwhelmed, message dropped")
			return nil // or return error to retry
		}
	}
}

// Consumer for the message broadcasting
func (hub *Hub) KafkaBroadcastConsumer() {
	// TODO: HOW TO STOP FROM OUTSIDE using ctx ?
	brokers := strings.Split(configs.Config.KAFKA_BROKERS, ",")

	channelBroadcastHandler := makeChannelBroadcastHandler(hub)
	hub.consumerManager.Add(brokers, "test-broadcast", "broadcast.channel-message.add", channelBroadcastHandler)

	// Start all
	go hub.consumerManager.Start()

	// Wait for shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// Graceful shutdown
	if err := hub.consumerManager.Stop(30 * time.Second); err != nil {
		logrus.Error(err)
	}
}
