package baseKafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// ConsumerManager runs multiple consumers with graceful shutdown
type ConsumerManager struct {
	name      string
	consumers []*Consumer
	wg        sync.WaitGroup
	cancel    context.CancelFunc
}

// New consumer manager for the kafka
func NewConsumerManager(name string) *ConsumerManager {
	return &ConsumerManager{
		name:      name,
		consumers: make([]*Consumer, 0),
	}
}

// Add registers a consumer but doesn't start it yet
func (cm *ConsumerManager) Add(brokers []string, groupID, topic string, handler func(*kafka.Message) error) {
	consumer := NewConsumer(brokers, groupID, topic, handler)
	cm.consumers = append(cm.consumers, consumer)
}

// Start runs all consumers in parallel
func (cm *ConsumerManager) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	cm.cancel = cancel
	logrus.Infof("Running `%s` kafka consumer manager...", cm.name)
	for _, consumer := range cm.consumers {
		cm.wg.Add(1)
		go func(c *Consumer) {
			defer cm.wg.Done()
			if err := c.Start(ctx); err != nil {
				logrus.WithError(err).Error("Consumer crashed")
			}
		}(consumer)
	}

}

// Stop gracefully shuts down all consumers
func (cm *ConsumerManager) Stop(timeout time.Duration) error {
	cm.cancel() // Signal shutdown

	done := make(chan struct{})
	go func() {
		cm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout for the `%s` kafka consumer mananger", cm.name)
	}
}
