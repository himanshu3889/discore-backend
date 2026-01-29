package memberStore

import (
	"database/sql"
	"discore/internal/modules/core/database"
	"discore/internal/modules/core/models"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Get the member based on userID and serverID
func GetUserServerMemember(ctx *gin.Context, userID snowflake.ID, serverID snowflake.ID) (*models.Member, error) {
	// NOTE: we have index on (user_id, server_id)
	query := `SELECT m.*
			  from members m
			  where m.user_id = $1 AND m.server_id = $2
			  LIMIT 1
			 `

	var member models.Member
	err := database.PostgresDB.GetContext(ctx, &member, query, userID, serverID)
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
