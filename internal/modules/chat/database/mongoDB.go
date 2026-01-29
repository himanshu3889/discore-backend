package database

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB client and database references (package-level)
var (
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
	once        sync.Once
)

// Initialize mongoDB establishes connection and creates indexes
func InitMongoDB() {
	once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Load .env file
		if err := godotenv.Load(".env"); err != nil {
			logrus.WithError(err).Fatal("Error loading .env file")
		}

		// Configure client
		username := os.Getenv("MONGODB_USERNAME")
		password := os.Getenv("MONGODB_PASSWORD")
		host := os.Getenv("MONGODB_HOST")
		database := os.Getenv("MONGODB_DATABASE")

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

		// Create indexes
		logrus.Info("Creating MongoDB indexes...")
		createIndexes(ctx)
		logrus.Info("MongoDB indexes created successfully")
	})
}

// createIndexes ensures optimal query performance
func createIndexes(ctx context.Context) {
	// Message indexes
	MongoDB.Collection("channel_messages").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// Main channel queries
			Keys: bson.D{
				{"channelId", 1}, // Filter by channel
				{"deleted", 1},   // Filter by deleted status
				{"_id", -1},      // Sort by createdAt (newest first)
			},
		},
		// {
		// 	// Channel + Member queries
		// 	Keys: bson.D{
		// 		{"channelId", 1}, // Filter by channel
		// 		{"deleted", 1},   // Filter by deleted status
		// 		{"memberID", 1},  // Then filter by member
		// 		{"_id", -1},      // Then sort by time
		// 	},
		// },
	})

	// Conversation unique constraint
	MongoDB.Collection("conversations").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{"memberOneID", 1},
			{"memberTwoID", 1},
		},
		Options: options.Index().SetUnique(true),
	})
	MongoDB.Collection("conversations").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{"memberTwoID", 1}}},
	})

	// DirectMessage indexes
	MongoDB.Collection("directMessages").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{"conversationID", 1}}},
		{Keys: bson.D{{"memberID", 1}}},
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
