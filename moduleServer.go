package main

import (
	"context"
	"discore/configs"
	baseApi "discore/internal/base/api"
	redisDatabase "discore/internal/base/infrastructure/redis"
	baseMiddlewares "discore/internal/base/middlewares"
	"discore/internal/base/utils"
	chatApi "discore/internal/modules/chat/api"
	chatDatabase "discore/internal/modules/chat/database"
	ChatkafkaService "discore/internal/modules/chat/services/kafka"
	coreApi "discore/internal/modules/core/api"
	clerkClient "discore/internal/modules/core/clients/clerk"
	"discore/internal/modules/core/database"
	websocketApi "discore/internal/modules/websocket/api"
	websocketApp "discore/internal/modules/websocket/application"
	websocketDatabase "discore/internal/modules/websocket/database"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ModuleServer struct {
	server *http.Server
	addr   string
}

func NewModuleServer(addr string) *ModuleServer {
	return &ModuleServer{
		addr: addr,
	}
}

func (s *ModuleServer) Initialize() {
	configs.InitializeConfigs()
	logrus.SetFormatter(&utils.LogrusColorFormatter{})

	router := gin.New()
	router.Use(baseMiddlewares.RequestIDMiddleware())
	router.Use(gin.Recovery())

	// Register routes
	baseGroup := router.Group("")
	baseApi.RegisterBaseRoutes(baseGroup)
	coreApi.RegisterCoreRoutes(baseGroup)
	chatApi.RegisterChatRoutes(baseGroup)
	websocketApi.RegisterWebsocketRoutes(baseGroup)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: router,
	}

	// Initialize all dependencies
	utils.InitSnowflake(2) // machineID = 2
	clerkClient.InitializeClerk()
	database.InitPostgresDB()
	chatDatabase.InitMongoDB()
	chatDatabase.InitPostgresDB()
	websocketDatabase.InitMongoDB()
	websocketDatabase.InitPostgresDB()
	websocketApp.InitializeHub(context.Background())
	redisDatabase.InitRedis()

	// Start Kafka consumer in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Panic recovered in Kafka consumer: %v", r)
			}
		}()
		ChatkafkaService.KafkaChatConsumer()
	}()
}

func (s *ModuleServer) Start() error {
	logrus.Infof("Starting module server on %s", s.addr)
	return s.server.ListenAndServe()
}

func (s *ModuleServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *ModuleServer) Run() {
	s.Initialize()

	// Start server in goroutine with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Panic recovered in module server: %v", r)
				// restart or handle gracefully
			}
		}()

		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Module server error: %s", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down module server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		logrus.Fatalf("Module server forced to shutdown: %v", err)
	}

	logrus.Info("Module server exited")
}
