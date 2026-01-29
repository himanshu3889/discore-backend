package memberStore

import (
	"context"
	"discore/internal/base/utils"
	"discore/internal/modules/core/database"
	"discore/internal/modules/core/models"
	"errors"

	"github.com/sirupsen/logrus"
)

// Create member in the server
func CreateMember(ctx context.Context, member *models.Member) error {
	const query = `INSERT INTO members 
				(id, role, user_id, server_id, created_at, updated_at) 
				values ($1, $2, $3, $4, NOW(), NOW()) 
				RETURNING *`

	member.ID = utils.GenerateSnowflakeID()
	err := database.PostgresDB.GetContext(ctx, member, query,
		member.ID,
		member.Role,
		member.UserID,
		member.ServerID,
	)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"role":      member.Role,
			"server_id": member.ServerID,
			"user_id":   member.UserID,
		}).WithError(err).Error("Failed to create member in database")
		return errors.New("Failed to create member for server")
	}
	return nil

}
