package channelStore

import (
	"database/sql"
	"discore/internal/modules/core/database"
	"discore/internal/modules/core/models"
	"errors"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func GetChannelByID(ctx *gin.Context, channelID snowflake.ID) (*models.Channel, error) {
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
		return nil, errors.New("Channel not found")
	}
	return &channel, nil

}
