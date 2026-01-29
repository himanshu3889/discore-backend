package serverStore

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

// Get the user own servers; Max limit is 10
func UserJoinedServers(ctx context.Context, user_id snowflake.ID) ([]*models.Server, error) {
	const query = `
        SELECT s.*
        FROM members m
        JOIN servers s ON m.server_id = s.id
        WHERE m.user_id = $1
        ORDER BY m.created_at ASC
		`
	var servers []*models.Server
	err := database.PostgresDB.SelectContext(ctx, &servers, query, user_id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*models.Server{}, nil // No server found, return nil without error
		}
		logrus.WithFields(logrus.Fields{
			"user_id": user_id,
		}).WithError(err).Errorf("Failed to find user servers on database")
		return servers, errors.New("failed to find user servers")
	}
	return servers, nil

}

// Get server by ID where user is a member
func GetServerByIDWithMembership(ctx context.Context, serverID snowflake.ID, userID snowflake.ID) (*models.Server, error) {
	const query = `
		SELECT s.*
		FROM servers s
		INNER JOIN members m ON s.id = m.server_id
		WHERE s.id = $1 AND m.user_id = $2
		LIMIT 1
	`

	var server models.Server
	err := database.PostgresDB.GetContext(ctx, &server, query, serverID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Server not found or user not a member
		}
		logrus.WithFields(logrus.Fields{
			"server_id": serverID,
			"user_id":   userID,
		}).WithError(err).Error("Failed to find server with membership")
		return nil, errors.New("failed to find server")
	}

	return &server, nil
}

// Get user first server
func UserFirstJoinedServer(ctx context.Context, user_id snowflake.ID) (*models.Server, error) {
	const query = `
        SELECT s.*
        FROM members m
        JOIN servers s ON m.server_id = s.id
        WHERE m.user_id = $1
        ORDER BY m.created_at ASC
        LIMIT 1`
	var server models.Server
	err := database.PostgresDB.GetContext(ctx, &server, query, user_id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No server found, return nil without error
		}
		logrus.WithFields(logrus.Fields{
			"user_id": user_id,
		}).WithError(err).Errorf("Failed to find user joined server on database")
		return &server, errors.New("failed to find user joined server")
	}
	return &server, nil

}

// Check if user own any server or not
func HasUserOwnAnyServer(ctx context.Context, user_id snowflake.ID) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM servers WHERE owner_id = $1)`
	var exists bool
	err := database.PostgresDB.GetContext(ctx, &exists, query, user_id)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id": user_id,
		}).WithError(err).Error("Database error during checking")
		return false, fmt.Errorf("Failed to check user own any servers")
	}
	return exists, nil

}

// Check if user own server or not
func HasUserOwnServer(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM servers WHERE id=$1 AND owner_id=$2)`

	var ok bool
	err := database.PostgresDB.GetContext(ctx, &ok, query,
		serverID,
		userID,
	)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id":   userID,
			"server_id": serverID,
		}).WithError(err).Error("Database error during checking user own server")
		return false, fmt.Errorf("failed to check user servers")
	}
	return ok, nil

}

// Retrieves server channels
func GetServerChannels(ctx context.Context, serverId snowflake.ID) ([]*models.Channel, error) {
	channelsQuery := `
        SELECT c.*
        FROM channels c
        WHERE c.server_id = $1
        ORDER BY c.created_at ASC
    `

	// Populate server.Channels slice
	var channels []*models.Channel
	err := database.PostgresDB.SelectContext(ctx, &channels, channelsQuery, serverId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*models.Channel{}, nil
		}
		logrus.WithFields(logrus.Fields{
			"server_id": serverId,
		}).WithError(err).Error("Failed to fetch channels from database")
		return nil, errors.New("Failed to get server channels")
	}

	return channels, nil
}

// Retrieves server members
func GetServerMembers(ctx context.Context, serverId snowflake.ID, limit int, afterSnowflake snowflake.ID) ([]*models.Member, error) {
	// Default limit 100 , max limit 200
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	var query string
	var args []interface{}

	// First page
	query = `
		SELECT *
		FROM members
		WHERE server_id = $1
		ORDER BY id ASC
		LIMIT $2
	`
	args = []interface{}{serverId, limit}

	if afterSnowflake > 0 {
		// Get members created after this afterSnowflake ID
		query = `
            SELECT *
            FROM members
            WHERE server_id = $1 AND id > $2
            ORDER BY id ASC
            LIMIT $3
        `
		args = []interface{}{serverId, afterSnowflake, limit}
	}

	// Populate server.Channels slice
	var members []*models.Member
	err := database.PostgresDB.SelectContext(ctx, &members, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*models.Member{}, nil
		}
		logrus.WithFields(logrus.Fields{
			"server_id": serverId,
		}).WithError(err).Error("Failed to fetch server members from database")
		return nil, errors.New("Failed to get server members")
	}

	// Get total count once
	// var total int
	// err = database.PostgresDB.GetContext(ctx, &total,
	// 	"SELECT COUNT(*) FROM members WHERE server_id = $1", serverId)
	// if err != nil {
	// 	return nil, err
	// }

	return members, nil
}
