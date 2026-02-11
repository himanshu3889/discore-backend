package accountStore

import (
	"context"
	"discore/internal/gateway/authenticationService/database"
	"discore/internal/gateway/authenticationService/models"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
)

// Return the user by id if found
func GetUserByID(ctx context.Context, userID snowflake.ID) (*models.User, error) {
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
		return nil, err
	}

	return &user, nil
}

// Return the user by the username if found
func GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
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
		return nil, err
	}

	return &user, nil
}

// Return the user by the email if found
func GetUserByEmail(ctx context.Context, email string) (*models.User, error) {

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
		return nil, err
	}

	return &user, nil
}

func GetAllUsers(ctx context.Context) ([]*models.User, error) {
	var users []*models.User
	query := `
		SELECT *
		FROM users
		ORDER BY id
	`

	err := database.PostgresDB.SelectContext(ctx, &users, query)
	if err != nil {
		logrus.WithError(err).Error("Failed to fetch user by email")
		return nil, err
	}

	return users, nil
}
