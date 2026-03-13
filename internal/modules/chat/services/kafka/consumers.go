package ChatkafkaService

import (
	"context"
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

func setupChatConsumers(ctx context.Context, brokers []string) *baseKafka.ConsumerManager {
	manager := baseKafka.NewConsumerManager("chat")
	kafkaProducer := baseKafka.NewProducer(brokers) // TODO: WHY TO CLOSE ?

	// Add multiple consumers
	cfg := baseKafka.ConsumerConfig{
		Brokers:        brokers,
		GroupID:        "bulk-test-add-2",
		Topic:          "channel-message.add",
		AutoCommit:     false, // no auto commit; if issue then in dlq then commit
		EnableBatching: true,  // batching here
		BatchSize:      100,
		BatchTimeout:   1000 * time.Millisecond,
		StartOffset:    kafka.LastOffset,
	}
	channelMessagesHandler := MakeChannelMessagesHandler(kafkaProducer)
	dlqHandler := MakeChannelMessagesDLQ(ctx, kafkaProducer)
	manager.Add(cfg, nil, channelMessagesHandler, dlqHandler)

	return manager
}

func KafkaChatConsumer() {
	ctx := context.Background()
	brokers := strings.Split(configs.Config.KAFKA_BROKERS, ",")
	manager := setupChatConsumers(ctx, brokers)

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
