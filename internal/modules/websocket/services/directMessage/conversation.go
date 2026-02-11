package directmessageService

import (
	"context"
	"discore/internal/base/utils"
	"discore/internal/modules/websocket/models"
	directmessage "discore/internal/modules/websocket/store/directMessage"
	"encoding/json"

	"github.com/bwmarrin/snowflake"
)

func SendDirectMessage(rawMessage *json.RawMessage, userID snowflake.ID) (*models.DirectMessage, error) {
	ctx := context.Background()
	// Step 1: Bind JSON into struct
	var msg models.DirectMessage
	if err := json.Unmarshal(*rawMessage, &msg); err != nil {
		return nil, err
	}

	msgID := utils.GenerateSnowflakeID()
	msg.ID = msgID
	msg.UserID = userID

	err := directmessage.CreateDirectMessage(ctx, &msg)

	return &msg, err
}
