package baseKafka

import (
	"discore/internal/base/utils"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/segmentio/kafka-go"
)

// Parse all headers at once into a struct
type MessageMetadata struct {
	ID        snowflake.ID `json:"ID"`
	Timestamp time.Time    `json:"timestamp"`
	UserID    snowflake.ID `json:"userID"`
}

func ParseKafkaMessageHeaders(msg *kafka.Message) *MessageMetadata {
	var meta MessageMetadata

	for _, h := range msg.Headers {
		switch h.Key {
		case "ID":
			meta.ID, _ = utils.ValidSnowflakeID(string(h.Value))
		case "timestamp":
			meta.Timestamp, _ = time.Parse(time.RFC3339, string(h.Value))
		case "userID":
			meta.UserID, _ = utils.ValidSnowflakeID(string(h.Value))
		}
	}

	return &meta
}
