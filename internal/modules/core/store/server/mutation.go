package serverStore

import (
	"context"
	"database/sql"
	"discore/internal/base/utils"
	"discore/internal/modules/core/database"
	"discore/internal/modules/core/models"
	coreUtils "discore/internal/modules/core/utils"
	"errors"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

func CreateServer(ctx context.Context, server *models.Server) error {
	const query = `INSERT INTO servers 
		(id, name, image_url, owner_id, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, NOW(), NOW()) 
		RETURNING *`
	server.ID = utils.GenerateSnowflakeID()
	if err := database.PostgresDB.GetContext(ctx, server, query,
		server.ID,
		server.Name,
		server.ImageUrl,
		server.OwnerID,
	); err != nil {
		logrus.WithFields(logrus.Fields{
			"server_name": server.Name,
			"owner_id":    server.OwnerID,
			"image_url":   server.ImageUrl,
		}).WithError(err).Error("Failed to create server in database")
		return fmt.Errorf("Failed to create server for user")
	}
	return nil

}

func UpdateServerNameImage(ctx context.Context, server *models.Server) error {
	const query = `
        UPDATE servers 
        SET name = $1, image_url= $2, updated_at = NOW()
        WHERE id = $3
        RETURNING *`

	// Update only allowed fields, return everything
	err := database.PostgresDB.GetContext(ctx, server, query,
		server.Name,
		server.ImageUrl,
		server.ID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			logrus.WithFields(logrus.Fields{
				"server_id": server.ID,
				"name":      server.Name,
				"imageUrl":  server.ImageUrl,
			}).Warn("Server not found for update")
			return fmt.Errorf("server not found")
		}
		logrus.WithFields(logrus.Fields{
			"server_id": server.ID,
			"name":      server.Name,
			"imageUrl":  server.ImageUrl,
		}).WithError(err).Error("Unable to update server due to database error")
		return errors.New("Unable to update server")
	}

	return nil
}

// Create server invite for the user; max attempts 3
func CreateServerInvite(ctx context.Context, serverInvite *models.ServerInvite) error {
	const query = `INSERT INTO server_invites 
	(code, server_id, created_by, max_uses, expires_at, created_at) 
	VALUES ($1, $2, $3, $4, $5, NOW()) 
	RETURNING *`

	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		serverInvite.Code = coreUtils.GenerateInviteCode()

		lastErr = database.PostgresDB.GetContext(ctx, serverInvite, query,
			serverInvite.Code,
			serverInvite.ServerID,
			serverInvite.CreatedBy,
			serverInvite.MaxUses,
			serverInvite.ExpiresAt,
		)

		if lastErr == nil {
			return nil
		}

		if !coreUtils.IsDBUniqueViolationError(lastErr) {
			logrus.WithFields(logrus.Fields{
				"server_id":  serverInvite.ServerID,
				"created_by": serverInvite.CreatedBy,
				"max_uses":   serverInvite.MaxUses,
				"attempt":    attempt,
			}).WithError(lastErr).Error("Failed to create server invite in database")
			return fmt.Errorf("Failed to create server invite for user")
		}

		logrus.WithFields(logrus.Fields{
			"server_id":  serverInvite.ServerID,
			"created_by": serverInvite.CreatedBy,
			"max_uses":   serverInvite.MaxUses,
			"attempt":    attempt,
		}).Debug("Invite code collision, retrying")
	}

	logrus.WithFields(logrus.Fields{
		"server_id":  serverInvite.ServerID,
		"created_by": serverInvite.CreatedBy,
		"max_uses":   serverInvite.MaxUses,
	}).WithError(lastErr).Error("Failed to create server invite after 5 attempts")
	return fmt.Errorf("Failed to create server invite for user: too many collisions")
}

// User server invite
func UseServerInvite(ctx context.Context, code string) error {
	result, err := database.PostgresDB.ExecContext(ctx,
		`UPDATE server_invites 
         SET used_count = used_count + 1 
         WHERE code = $1 
           AND (max_uses IS NULL OR used_count < max_uses)
           AND (expires_at IS NULL OR expires_at > NOW())`, // expiration check
		code,
	)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"server_invite_code": code,
		}).WithError(err).Error("Failed to use server invite")
		return errors.New("Failed to use server invite")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logrus.WithFields(logrus.Fields{
			"server_invite_code": code,
		}).WithError(err).Error("Server invite is invalid, expired, or maxed out")
		return errors.New("Server invite is invalid, expired, or maxed out")
	}
	return nil
}

// Accept the server invite and create memember; if already a member then don't consume invite
func AcceptServerInviteAndCreateMember(ctx context.Context, userID snowflake.ID, code string) (*snowflake.ID, error) {
	// Get the server ID from the code
	tx, err := database.PostgresDB.BeginTxx(ctx, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"invite_code": code,
		}).WithError(err).Error("Failed to begin the server invite accept transaction")
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if transaction failed

	// Find the server_id
	var serverID snowflake.ID
	serverQuery := `SELECT server_id FROM server_invites WHERE code=$1`
	err = tx.GetContext(ctx, &serverID, serverQuery, code)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"invite_code": code,
		}).WithError(err).Errorf("Failed to query the server invite in database")
		return nil, fmt.Errorf("Failed to accept the invite")
	}

	// Try to create member → Consume invite → Commit

	// Try to create member FIRST
	member := &models.Member{
		ID:       utils.GenerateSnowflakeID(),
		UserID:   userID,
		ServerID: serverID,
		Role:     "GUEST",
	}

	// Returning *
	insertQuery := `INSERT INTO members (id, role, user_id, server_id, created_at, updated_at)
                    VALUES ($1, $2, $3, $4, NOW(), NOW())
                    RETURNING *`

	err = tx.GetContext(ctx, member, insertQuery,
		member.ID,
		member.Role,
		member.UserID,
		member.ServerID)
	if err != nil {
		if coreUtils.IsDBUniqueViolationError(err) {
			return &serverID, nil // errors.New("Already a member of this server")
		}
		logrus.WithFields(logrus.Fields{
			"role":      member.Role,
			"server_id": member.ServerID,
			"user_id":   member.UserID,
		}).WithError(err).Error("Failed to create member in database")
		return nil, errors.New("Failed to create member for server")
	}

	updateQuery := `UPDATE server_invites
					SET used_count = used_count + 1
					WHERE code=$1
						AND (max_uses IS NULL OR used_count < max_uses)
						AND (expires_at IS NULL OR expires_at > NOW())`

	result, err := tx.ExecContext(ctx, updateQuery, code)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"invite_code": code,
		}).WithError(err).Errorf("Failed to update invite in database")
		return nil, fmt.Errorf("Unable to accept the invite")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logrus.WithFields(logrus.Fields{
			"invite_code": code,
		}).WithError(err).Errorf("invite is invalid, expired, or maxed out")
		return nil, errors.New("invite is invalid, expired, or maxed out")
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		logrus.WithFields(logrus.Fields{
			"invite_code": code,
		}).WithError(err).Errorf("failed to commit server invite accept transaction")
		return nil, fmt.Errorf("Unable to accept the invite")
	}

	return &member.ServerID, nil
}
