package memberStore

import (
	"context"

	"github.com/himanshu3889/discore-backend/base/databases"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	"github.com/himanshu3889/discore-backend/base/models"
	"github.com/himanshu3889/discore-backend/base/utils"

	"github.com/sirupsen/logrus"
)

// Create member in the server
func CreateMember(ctx context.Context, member *models.Member) *appError.Error {
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
		return appError.NewInternal("Failed to create member for server")
	}
	return nil

}
