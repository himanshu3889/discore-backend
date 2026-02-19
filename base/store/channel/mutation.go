package channelStore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/himanshu3889/discore-backend/base/databases"
	"github.com/himanshu3889/discore-backend/base/models"
	"github.com/himanshu3889/discore-backend/base/utils"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Create channel in the server
func CreateChannel(ctx context.Context, channel *models.Channel) error {
	const query = `INSERT INTO channels 
		(id, name, type, creator_id, server_id, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
		RETURNING *`

	channel.ID = utils.GenerateSnowflakeID()
	err := database.PostgresDB.GetContext(ctx, channel, query,
		channel.ID,
		channel.Name,
		channel.Type,
		channel.CreatorID,
		channel.ServerID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"channel_type": channel.Type,
			"server_id":    channel.ServerID,
			"user_id":      channel.CreatorID,
		}).WithError(err).Error("Failed to create channel for server")
		return errors.New("Failed to create channel for the server")
	}
	return nil
}

// Update channel name by the channel id
func UpdateChannelNameType(ctx context.Context, channel *models.Channel) error {
	const query = `
        UPDATE channels 
		SET name = $1, type =$2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *
		`

	// Update only allowed fields, return everything
	err := database.PostgresDB.GetContext(ctx, channel, query,
		channel.Name,
		channel.Type,
		channel.ID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			logrus.WithFields(logrus.Fields{
				"channel_id": channel.ID,
			}).Warn("Channel not found for update")
			return fmt.Errorf("Channel not found")
		}
		logrus.WithFields(logrus.Fields{
			"channel_id":   channel.ID,
			"channel_name": channel.Name,
		}).WithError(err).Error("Failed to update channel in database")
		return errors.New("Failed to update channel")
	}

	return nil
}

// Soft delete the channel
func SoftDeleteChannelById(ctx context.Context, channelID snowflake.ID) (*models.Channel, error) {
	// Soft Delete
	const query = `
        UPDATE channels 
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING *
		`
	//  `DELETE FROM channels WHERE id = $1 RETURNING *`

	// Update only allowed fields, return everything
	var channel models.Channel
	err := database.PostgresDB.QueryRowContext(ctx, query,
		channelID,
	).Scan(&channel.ID, &channel.Name, &channel.Type, &channel.CreatorID, &channel.ServerID, &channel.CreatedAt, &channel.UpdatedAt, &channel.DeletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			logrus.WithFields(logrus.Fields{
				"channel_id": channelID,
			}).Warn("Channel not found for soft delete")
			return nil, fmt.Errorf("Channel not found")
		}
		logrus.WithFields(logrus.Fields{
			"channel_id": channelID,
		}).WithError(err).Error("Failed to soft delete channel in database")
		return nil, errors.New("Failed to soft delete channel")
	}
	return &channel, err
}

// Permanently delete the channel
func HardDeleteChannelById(ctx context.Context, channelID snowflake.ID) (*models.Channel, error) {
	const query = `DELETE FROM channels 
				WHERE id = $1 
				RETURNING *`

	// Update only allowed fields, return everything
	var channel models.Channel
	err := database.PostgresDB.QueryRowContext(ctx, query,
		channelID,
	).Scan(&channel.ID, &channel.Name, &channel.Type, &channel.CreatorID, &channel.ServerID, &channel.CreatedAt, &channel.UpdatedAt, &channel.DeletedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			logrus.WithFields(logrus.Fields{
				"channel_id": channelID,
			}).Warn("Channel not found for soft delete")
			return nil, fmt.Errorf("Channel not found")
		}
		logrus.WithFields(logrus.Fields{
			"channel_id": channelID,
		}).WithError(err).Error("Failed to soft delete channel in database")
		return nil, errors.New("Failed to soft delete channel")
	}
	return &channel, err
}
