package ChatkafkaService

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	baseKafka "github.com/himanshu3889/discore-backend/base/infrastructure/kafka"
	"github.com/himanshu3889/discore-backend/configs"

	"github.com/sirupsen/logrus"
)

func setupChatConsumers(brokers []string) *baseKafka.ConsumerManager {
	manager := baseKafka.NewConsumerManager("chat")
	kafkaProducer := baseKafka.NewProducer(brokers) // TODO: WHY TO CLOSE ?

	// Add multiple consumers
	channelMessageHandler := MakeChannelMessageHandler(kafkaProducer)
	manager.Add(brokers, "test-add", "channel-message.add", channelMessageHandler)

	return manager
}

func KafkaChatConsumer() {
	brokers := strings.Split(configs.Config.KAFKA_BROKERS, ",")
	manager := setupChatConsumers(brokers)

	// Start all
	go manager.Start()

	// Wait for shutdown signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	// Graceful shutdown
	if err := manager.Stop(30 * time.Second); err != nil {
		logrus.Error(err)
	}
}
