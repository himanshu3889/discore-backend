package main

import (
	"context"
	baseApi "discore/internal/base/api"
	redisDatabase "discore/internal/base/infrastructure/redis"
	"discore/internal/gateway"
	chatApi "discore/internal/modules/chat/api"
	messagingDatabase "discore/internal/modules/chat/database"
	coreApi "discore/internal/modules/core/api"
	clerkClient "discore/internal/modules/core/clients/clerk"
	"discore/internal/modules/core/database"
	websocketApi "discore/internal/modules/websocket/api"
	websocketApp "discore/internal/modules/websocket/application"
	"sync"

	baseMiddlewares "discore/internal/base/middlewares"
	"discore/internal/base/utils"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// NOTE: zap logger is fast, efficient

func StartModuleServer(wg *sync.WaitGroup) {
	defer wg.Done()

	err := godotenv.Load()
	if err != nil {
		logrus.Error("No .env file found, using system env vars")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensures cleanup if main exits

	// logrus formatting
	logrus.SetFormatter(&utils.LogrusColorFormatter{})

	router := gin.New()

	// Middlewares
	router.Use(baseMiddlewares.CORSMiddleware())      // use the cors middleware
	router.Use(baseMiddlewares.RequestIDMiddleware()) // Request ID attach
	router.Use(gin.Recovery())                        // Recover panic middleware

	// All routes
	baseGroup := router.Group("")
	// Base
	baseApi.RegisterBaseRoutes(baseGroup)
	// Core
	coreApi.RegisterCoreRoutes(baseGroup)
	// Chat
	chatApi.RegisterChatRoutes(baseGroup)
	// Websocket
	websocketApi.RegisterWebsocketRoutes(baseGroup)

	// server configuration
	const addr string = ":8080"
	const machineID = 2
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Run server in goroutine
	go func() {
		logrus.Infof("Starting server on %s", addr)
		utils.InitSnowflake(machineID)  // Initialize snowflake
		clerkClient.InitializeClerk()   // Initialize clerk
		database.InitPostgresDB()       // Initialize postgres
		messagingDatabase.InitMongoDB() // Initialize mongodb
		// go websocketApp.HandleWebsocketMessages(ctx) // go routine for the websockets
		websocketApp.InitializeHub(ctx) // websocket
		//
		redisDatabase.InitRedis() // Initialize redis

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("listen: %s\n", err)
		}

	}()

	// Graceful shutdown
	func() {
		quit := make(chan os.Signal, 1) // make a channel to receive the signal

		signal.Notify(quit, os.Interrupt) // notify the channel when the signal is received
		<-quit                            // wait for the signal to be received
		logrus.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // context with timeout
		defer cancel()                                                           // cancel the context when the function returns
		if err := srv.Shutdown(ctx); err != nil {                                // shutdown the server
			logrus.Fatalf("Server forced to shutdown: %v", err) // log the error if the server fails to shutdown
		}

		logrus.Info("Server exiting") // log the server exiting
	}()
}

func StartGatewayServer(wg *sync.WaitGroup) {
	defer wg.Done()

	err := godotenv.Load()
	if err != nil {
		logrus.Error("No .env file found, using system env vars")
	}

	// logrus formatting
	logrus.SetFormatter(&utils.LogrusColorFormatter{})

	gateway := gateway.NewGateway()

	// server configuration
	const addr string = ":8090"
	// const machineID = 1
	srv := &http.Server{
		Addr:         addr,
		Handler:      gateway.GetEngine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run server in goroutine
	go func() {
		logrus.Infof("Starting gateway server on %s", addr)
		// basePrometheus.InitPrometheus() // Prometheus Initialization
		redisDatabase.InitRedis() // Redis initialization

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("listen: %s\n", err)
		}

	}()

	// Graceful shutdown
	func() {
		quit := make(chan os.Signal, 1) // make a channel to receive the signal

		signal.Notify(quit, os.Interrupt) // notify the channel when the signal is received
		<-quit                            // wait for the signal to be received
		logrus.Info("Shutting down gateway server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // context with timeout
		defer cancel()                                                           // cancel the context when the function returns
		if err := srv.Shutdown(ctx); err != nil {                                // shutdown the server
			logrus.Fatalf("Gateway server forced to shutdown: %v", err) // log the error if the server fails to shutdown
		}

		logrus.Info("Gateway server exiting") // log the server exiting
	}()
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go StartModuleServer(&wg)
	go StartGatewayServer(&wg)

	wg.Wait() // Now properly waits for BOTH to receive shutdown signals
}
