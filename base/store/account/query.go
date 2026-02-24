package accountStore

import (
	"context"

	"github.com/himanshu3889/discore-backend/base/databases"
	"github.com/himanshu3889/discore-backend/base/lib/appError"
	"github.com/himanshu3889/discore-backend/base/models"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Return the user by id if found
func GetUserByID(ctx context.Context, userID snowflake.ID) (*models.User, *appError.Error) {
	var user models.User

	query := `
		SELECT *
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	err := database.PostgresDB.GetContext(ctx, &user, query, userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch user by ID")
		return nil, appError.NewInternal("Failed to fetch user")
	}

	return &user, nil
}

// Return the user by the username if found
func GetUserByUsername(ctx context.Context, username string) (*models.User, *appError.Error) {
	var user models.User

	query := `
		SELECT *
		FROM users
		WHERE username = $1
		LIMIT 1
	`

	err := database.PostgresDB.GetContext(ctx, &user, query, username)
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch user by username")
		return nil, appError.NewInternal("Failed to fetch user ")
	}

	return &user, nil
}

// Return the user by the email if found
func GetUserByEmail(ctx context.Context, email string) (*models.User, *appError.Error) {

	var user models.User

	query := `
		SELECT *
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	err := database.PostgresDB.GetContext(ctx, &user, query, email)
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch user by email")
		return nil, appError.NewInternal("Failed to fetch user ")
	}

	return &user, nil
}
