package accountStore

import (
	"context"
	"database/sql"
	"discore/internal/base/utils"
	"discore/internal/gateway/authenticationService/database"
	"discore/internal/gateway/authenticationService/models"
	coreUtils "discore/internal/modules/core/utils"
	"errors"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Create a new user
func CreateUser(ctx context.Context, user *models.User) error {
	const queryUserInsert = `
		INSERT INTO users (id, username, email, password, name, image_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, Now(), Now())
		RETURNING *
	`

	user.ID = utils.GenerateSnowflakeID()
	if err := database.PostgresDB.GetContext(ctx, user, queryUserInsert,
		user.ID,
		user.Email,
		user.Email,
		user.Password,
		user.Name,
		user.ImageUrl,
	); err != nil {
		if coreUtils.IsDBUniqueViolationError(err) {
			logrus.WithField("email", user.Email).Warn("User already exists in database")
			return errors.New("User already exists in database")
		}
		logrus.WithFields(logrus.Fields{
			"email":           user.Email,
			"password_length": len(user.Password),
			"image_url":       user.ImageUrl,
		}).WithError(err).Error("Failed to create user")
		return errors.New("Failed to create user")
	}
	return nil
}

// Create a new session for user in session
func CreateSession(ctx context.Context, session *models.UserSession) error {
	const querySessionInsert = `
        INSERT INTO user_sessions (id, user_id, refresh_token, device_info,
                                   ip_address, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING *`
	err := database.PostgresDB.GetContext(ctx, session, querySessionInsert,
		utils.GenerateSnowflakeID(),
		session.UserID,
		session.RefreshToken,
		session.DeviceInfo,
		session.IPAddress,
		session.ExpiresAt)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id": session.UserID,
		}).WithError(err).Error("Failed to create user session")
		return errors.New("Failed to create user session")
	}
	return nil
}

// Returns the session only if it belongs to the given user and is still valid.
func GetUserSessionByToken(ctx context.Context, userID snowflake.ID, token string) (*models.UserSession, error) {
	const querySessionGet = `
		SELECT *
		FROM user_sessions
		WHERE user_id = $1
		  AND refresh_token = $2
		  AND expires_at > NOW()`
	var s models.UserSession
	err := database.PostgresDB.GetContext(ctx, &s, querySessionGet, userID, token)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id": userID,
			"token":   token,
		}).Error("Unable to get user session token")
		return nil, fmt.Errorf("Unable to get user %s session token", userID)
	}
	return &s, nil
}

// Removes the exact session row for that user/token pair.
func DeleteUserSession(ctx context.Context, userID snowflake.ID, refreshToken string) error {
	const queryDeleteSession = `DELETE FROM user_sessions WHERE user_id = $1 AND refresh_token = $2`
	_, err := database.PostgresDB.ExecContext(ctx, queryDeleteSession, userID, refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			logrus.WithFields(logrus.Fields{
				"user_id": userID,
			}).Warn("session not found for delete")
			return fmt.Errorf("session not found")
		}
		return fmt.Errorf("Unable to delete user session")
	}
	return err
}

// Delete all user sessions logs the user out everywhere.
func DeleteUserAllSessions(ctx context.Context, userID snowflake.ID) error {
	const queryDeleteAllSessions = `DELETE FROM user_sessions WHERE user_id = $1`
	_, err := database.PostgresDB.ExecContext(ctx, queryDeleteAllSessions, userID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id": userID,
		}).WithError(err).Error("Unable to delete the user sessions")
		return fmt.Errorf("Unable to delete the user sessions")
	}
	return err
}
