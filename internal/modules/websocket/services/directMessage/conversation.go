package directmessageService

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/himanshu3889/discore-backend/base/models"
	directmessage "github.com/himanshu3889/discore-backend/base/store/directMessage"
	"github.com/himanshu3889/discore-backend/base/utils"

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

	appErr := directmessage.CreateDirectMessage(ctx, &msg)

	return &msg, errors.New(appErr.Message)
}
