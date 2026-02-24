package serverStore

import (
	"context"
	"database/sql"

	"github.com/himanshu3889/discore-backend/base/databases"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	"github.com/himanshu3889/discore-backend/base/models"
	"github.com/himanshu3889/discore-backend/base/utils"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Create server
func CreateServer(ctx context.Context, server *models.Server) *appError.Error {
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
		return appError.NewInternal("Failed to create server for user")
	}
	return nil

}

// Update the server name and image; FIXME: name or image ?
func UpdateServerNameImage(ctx context.Context, server *models.Server) *appError.Error {
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
			return appError.NewNotFound("Server not found")
		}
		logrus.WithFields(logrus.Fields{
			"server_id": server.ID,
			"name":      server.Name,
			"imageUrl":  server.ImageUrl,
		}).WithError(err).Error("Unable to update server due to database error")
		return appError.NewInternal("Unable to update server")
	}

	return nil
}

// Create server invite for the user; max attempts 3
func CreateServerInvite(ctx context.Context, serverInvite *models.ServerInvite) *appError.Error {
	const query = `INSERT INTO server_invites 
	(code, server_id, created_by, max_uses, expires_at, created_at) 
	VALUES ($1, $2, $3, $4, $5, NOW()) 
	RETURNING *`

	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		serverInvite.Code = utils.GenerateInviteCode()

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

		if !utils.IsDBUniqueViolationError(lastErr) {
			logrus.WithFields(logrus.Fields{
				"server_id":  serverInvite.ServerID,
				"created_by": serverInvite.CreatedBy,
				"max_uses":   serverInvite.MaxUses,
				"attempt":    attempt,
			}).WithError(lastErr).Error("Failed to create server invite in database")
			return appError.NewInternal("Failed to create server invite for user")
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
	return appError.NewInternal("Failed to create server invite for user")
}

// Get the server invite
func GetServerInvite(ctx context.Context, code string) (*models.ServerInvite, *appError.Error) {
	var serverInvite models.ServerInvite
	inviteQuery := `SELECT * FROM server_invites WHERE code=$1`
	err := database.PostgresDB.GetContext(ctx, &serverInvite, inviteQuery, code)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"invite_code": code,
		}).WithError(err).Errorf("Failed to query the server invite in database")
		return nil, appError.NewInternal("Failed to accept the invite")
	}
	return &serverInvite, nil
}

// Accept the server invite and create memember; if already a member then don't consume invite, return serverInvite
func CreateServerMember(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (*models.Member, *appError.Error) {
	// Try to create member
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

	err := database.PostgresDB.GetContext(ctx, member, insertQuery,
		member.ID,
		member.Role,
		member.UserID,
		member.ServerID)
	if err != nil {
		if utils.IsDBUniqueViolationError(err) {
			return member, nil // errors.New("Already a member of this server")
		}
		logrus.WithFields(logrus.Fields{
			"role":      member.Role,
			"server_id": member.ServerID,
			"user_id":   member.UserID,
		}).WithError(err).Error("Failed to create member in database")
		return nil, appError.NewInternal("Failed to create member for server")
	}

	return member, nil
}

// Use the server invite
func UseServerInvite(ctx context.Context, code string) *appError.Error {
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
		return appError.NewInternal("Failed to use server invite")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logrus.WithFields(logrus.Fields{
			"server_invite_code": code,
		}).WithError(err).Error("Server invite is invalid, expired, or maxed out")
		return appError.NewBadRequest("Server invite is invalid, expired, or maxed out")
	}
	return nil
}
