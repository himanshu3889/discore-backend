package memberStore

import (
	"context"
	"discore/internal/modules/chat/database"
	"discore/internal/modules/chat/models"
	"errors"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// GetMembersBatchWithUsers fetches members with their user data in one query
func GetMembersBatchWithUsers(ctx context.Context, userIDs []snowflake.ID, serverID snowflake.ID) (map[snowflake.ID]*models.Member, error) {
	if len(userIDs) == 0 {
		return map[snowflake.ID]*models.Member{}, nil
	}
	if len(userIDs) > 100 {
		return nil, errors.New("max user batching is 100")
	}

	query := `
        SELECT 
            m.id, m.role, m.user_id, m.server_id, m.created_at, m.updated_at, m.deleted_at,
            u.id AS "user.id", u.name AS "user.name", u.username AS "user.username", u.email AS "user.email",
            u.image_url AS "user.image_url"
        FROM members m
        JOIN users u ON m.user_id = u.id
        WHERE m.user_id = ANY($1) AND m.server_id = $2 
    `

	var members []*models.Member
	err := database.PostgresDB.SelectContext(ctx, &members, query, pq.Array(userIDs), serverID)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_ids":  len(userIDs),
			"server_id": serverID,
		}).WithError(err).Error("Failed to fetch members batch with users")
		return nil, errors.New("failed to fetch members with users")
	}

	// Convert to map keyed by UserID for O(1) lookups
	memberMap := make(map[snowflake.ID]*models.Member, len(members))
	for _, member := range members {
		memberMap[member.UserID] = member
	}

	return memberMap, nil
}

// Has user is member of the server
func HasUserServerMember(ctx context.Context, userID snowflake.ID, serverID snowflake.ID) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1
								from members
								where user_id = $1 AND server_id = $2)
								`
	var exists bool
	err := database.PostgresDB.GetContext(ctx, &exists, query,
		userID,
		serverID,
	)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id":   userID,
			"server_id": serverID,
		}).WithError(err).Error("Database error during checking")
		return false, fmt.Errorf("Failed to check user own any servers")
	}
	return exists, nil

}
