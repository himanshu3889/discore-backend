package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/himanshu3889/discore-backend/configs"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// PostgresDB is now a sqlx connection
var (
	PostgresDB *sqlx.DB
	once       sync.Once
)

// Initialize postgres database
func InitPostgresDB() {
	once.Do(func() {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s sslmode=disable",
			configs.Config.POSTGRES_HOST,
			configs.Config.POSTGRES_USER,
			configs.Config.POSTGRES_PASSWORD,
			configs.Config.POSTGRES_DB,
		)

		var err error
		PostgresDB, err = sqlx.Connect("postgres", dsn)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to connect to PostgreSQL")
		}

		// Verify connection
		if err := PostgresDB.Ping(); err != nil {
			logrus.WithError(err).Fatal("Failed to ping PostgreSQL")
		}

		// Set pool limits (adjust as needed)
		PostgresDB.SetMaxOpenConns(25)
		PostgresDB.SetMaxIdleConns(25)
		PostgresDB.SetConnMaxLifetime(5 * time.Minute)

		logrus.Info("Postgres Database connected successfully")

		// You must run migrations manually or use a tool like golang-migrate
		logrus.Info("Remember to run your SQL migration scripts!")
	})
}
