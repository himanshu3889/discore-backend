package coreKafkaService

import (
	"context"
	"encoding/json"
	"errors"

	baseDebezium "github.com/himanshu3889/discore-backend/base/infrastructure/debezium"
	baseKafka "github.com/himanshu3889/discore-backend/base/infrastructure/kafka"
	serverStore "github.com/himanshu3889/discore-backend/base/store/server"
	"github.com/segmentio/kafka-go"
)

type memberDebezium struct {
	ID int64 `json:"id"`
	// Map the JSON key explicitly here for the worker
	InviteCodeUsed *string `json:"invite_code_used"`
	Role           string  `json:"role"`
}

// MakeChannelMessageHandler creates a closure to handle a batch of Kafka messages
func MakeInvitedMembersHandler(ctx context.Context, producer *baseKafka.KafkaProducer) func([]*kafka.Message) (error, []*kafka.Message) {
	return func(messages []*kafka.Message) (error, []*kafka.Message) {

		// map[inviteCode]count
		inviteUsage := make(map[string]int)

		for _, msg := range messages {
			var event baseDebezium.DebeziumEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				continue // Skip message
			}

			// if operation is Create
			if event.Op == "c" {
				var member memberDebezium
				if err := json.Unmarshal(event.After, &member); err != nil {
					continue
				}

				// If invite_code_used is not null, increment the count
				if member.InviteCodeUsed != nil && *member.InviteCodeUsed != "" {
					inviteUsage[*member.InviteCodeUsed]++
				}
			}
		}

		// Apply the counts to your store
		for code, count := range inviteUsage {
			appErr := serverStore.UseServerInvite(ctx, code, count) // Assuming UseServerInvite takes code and count
			if appErr != nil {
				// If DB fails, return error and the messages to retry
				// TODO: use the dlq here; as if we retried we actually run of duplicates
				return errors.New(appErr.Message), messages
			}
		}

		return nil, nil
	}
}
