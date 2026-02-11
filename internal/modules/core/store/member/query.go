package memberStore

import (
	"context"
	"database/sql"
	"discore/internal/modules/core/database"
	"discore/internal/modules/core/models"
	"errors"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Get the member based on userID and serverID
func GetUserServerMemember(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (*models.Member, error) {
	// NOTE: we have index on (user_id, server_id)
	query := `SELECT *
			  from members
			  where user_id = $1 AND server_id = $2
			  LIMIT 1
			 `

	var member models.Member
	err := database.PostgresDB.GetContext(ctx, &member, query,
		userID,
		serverID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("Server member not found!")
		}
		logrus.WithFields(logrus.Fields{
			"user_id":   userID,
			"server_id": serverID,
		}).WithError(err).Errorf("Failed to find server member on database")
		return nil, errors.New("Failed to find server member")
	}
	return &member, nil
}

// Has user is member of the server
func HasUserServerMember(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (bool, error) {
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
			"user_id": userID,
		}).WithError(err).Error("Database error during checking")
		return false, fmt.Errorf("Failed to check user own any servers")
	}
	return exists, nil

}
