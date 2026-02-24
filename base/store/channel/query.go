package channelStore

import (
	"database/sql"
	"errors"

	"github.com/himanshu3889/discore-backend/base/databases"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	"github.com/himanshu3889/discore-backend/base/models"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Get channel by ID
func GetChannelByID(ctx *gin.Context, channelID snowflake.ID) (*models.Channel, *appError.Error) {
	query := `
		SELECT *
		FROM channels c
		WHERE c.id = $1 AND deleted_at IS NULL
		LIMIT 1
		`

	var channel models.Channel
	err := database.PostgresDB.GetContext(ctx, &channel, query, channelID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			logrus.WithFields(logrus.Fields{
				"channel_id": channelID,
			}).WithError(err).Error("Failed to fetch channels from database")
		}
		return nil, appError.NewNotFound("Channel not found")
	}
	return &channel, nil

}
