package websocketApp

import (
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	baseKafka "github.com/himanshu3889/discore-backend/base/infrastructure/kafka"
	"github.com/himanshu3889/discore-backend/configs"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Make handler for channel broadcasting
func makeChannelBroadcastHandler(hub *Hub) func(*kafka.Message) (error, *kafka.Message) {
	return func(msg *kafka.Message) (error, *kafka.Message) {
		// logrus.Infof("RAW JSON in broadcasting: %s\n", string(msg.Value))

		rawData := &json.RawMessage{}
		err := json.Unmarshal(msg.Value, rawData)
		if err != nil {
			return nil, nil
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
			return nil, nil
		}

		var socketMessage = &BroadcastRequest{
			Event:         EventType(msgTopic),
			Room:          room,
			Data:          rawData,
			PipelineStart: kafkaMetadata.IngestTime,
		}

		select {
		case roomState.outBuffer <- socketMessage: // send message to room
			return nil, nil
		case <-time.After(5 * time.Second):
			// Optional: timeout if workers stuck too long
			logrus.Warn("Workers overwhelmed, message dropped")
			return nil, nil
		}
	}
}

// Consumer for the message broadcasting
func (hub *Hub) KafkaBroadcastConsumer() {
	// TODO: HOW TO STOP FROM OUTSIDE using ctx ?
	brokers := strings.Split(configs.Config.KAFKA_BROKERS, ",")

	channelBroadcastHandler := makeChannelBroadcastHandler(hub)
	cfg := baseKafka.ConsumerConfig{
		Brokers:     brokers,
		GroupID:     "test-broadcast-2",
		Topic:       "broadcast.channel-message.add",
		AutoCommit:  false, // no auto commit
		StartOffset: kafka.LastOffset,
	}
	hub.consumerManager.Add(cfg, channelBroadcastHandler, nil, nil)

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
