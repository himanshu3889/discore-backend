package database

import (
	"context"
	"discore/configs"
	"fmt"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB client and database references (package-level)
var (
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
	mongoOnce   sync.Once
)

// Initialize mongoDB establishes connection and creates indexes
func InitMongoDB() {
	mongoOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Load .env file
		if err := godotenv.Load(".env"); err != nil {
			logrus.WithError(err).Fatal("Error loading .env file")
		}

		// Configure client
		username := configs.Config.MONGODB_USERNAME
		password := configs.Config.MONGODB_PASSWORD
		host := configs.Config.MONGODB_HOST
		database := configs.Config.MONGODB_DATABASE

		uri := fmt.Sprintf("mongodb://%s:%s@%s:27017/%s?authSource=%s",
			username, password, host, database, database)

		logrus.Info(uri)

		clientOptions := options.Client().ApplyURI(uri).
			SetTimeout(10 * time.Second).
			SetMaxPoolSize(10).
			SetMinPoolSize(2)

		var err error
		MongoClient, err = mongo.Connect(ctx, clientOptions)
		if err != nil {
			logrus.WithError(err).Fatal("MongoDB connection failed")
		}

		// Verify connection
		if err := MongoClient.Ping(ctx, nil); err != nil {
			logrus.WithError(err).Fatal("MongoDB ping failed")
		}

		logrus.Info("MongoDB connected successfully")

		// Get database reference
		MongoDB = MongoClient.Database(database)

	})
}

// DisconnectMongoDB closes the connection gracefully
func DisconnectMongoDB() {
	if MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := MongoClient.Disconnect(ctx); err != nil {
			logrus.WithError(err).Error("Error disconnecting from MongoDB")
		} else {
			logrus.Info("MongoDB disconnected")
		}
	}
}
