package serverStore

import (
	"context"
	"discore/internal/modules/websocket/database"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Has user is member of the server
func HasUserServerMember(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (bool, error) {
	return true, nil
	const query = `SELECT EXISTS(SELECT 1
								from members
								where user_id = $1 AND server_id = $2)
								`
	var exists bool
	err := database.PostgresDB.GetContext(ctx, &exists, query,
		userID,
		serverID,
	)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id":   userID,
			"server_id": serverID,
		}).WithError(err).Error("Database error during checking")
		return false, fmt.Errorf("Failed to check user own any servers")
	}
	return exists, nil

}
