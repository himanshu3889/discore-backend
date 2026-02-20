package servers

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	baseApi "github.com/himanshu3889/discore-backend/base/api"
	clerkClient "github.com/himanshu3889/discore-backend/base/clients/clerk"
	"github.com/himanshu3889/discore-backend/base/databases"
	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/middlewares"
	"github.com/himanshu3889/discore-backend/base/utils"
	"github.com/himanshu3889/discore-backend/configs"
	app "github.com/himanshu3889/discore-backend/internal/modules"
	chatApi "github.com/himanshu3889/discore-backend/internal/modules/chat/api"
	ChatkafkaService "github.com/himanshu3889/discore-backend/internal/modules/chat/services/kafka"
	coreApi "github.com/himanshu3889/discore-backend/internal/modules/core/api"
	websocketApi "github.com/himanshu3889/discore-backend/internal/modules/websocket/api"
	websocketApp "github.com/himanshu3889/discore-backend/internal/modules/websocket/application"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	// logrus.SetFormatter(&utils.LogrusColorFormatter{})

	router := gin.New()
	router.Use(middlewares.MetricsMiddleware()) // Captures metrics
	router.Use(middlewares.RequestIDMiddleware())
	router.Use(gin.Recovery())

	// Register routes
	baseGroup := router.Group("")
	baseApi.RegisterBaseRoutes(baseGroup)
	coreApi.RegisterCoreRoutes(baseGroup)
	chatApi.RegisterChatRoutes(baseGroup)
	websocketApi.RegisterWebsocketRoutes(baseGroup)
	// Prometheus
	baseGroup.GET("/metrics", gin.WrapH(promhttp.Handler()))

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: router,
	}

	// Initialize all dependencies
	utils.InitSnowflake(2) // machineID = 2
	clerkClient.InitializeClerk()
	database.InitPostgresDB()
	database.InitMongoDB()
	database.InitPostgresDB()
	database.InitMongoDB()
	database.InitPostgresDB()
	websocketApp.InitializeHub(context.Background())
	redisDatabase.InitRedis()

	// Set the app modules app state
	// Connect to authentication server using grpc
	backoffConfig := grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  100 * time.Millisecond, // Start retrying very quickly
			Multiplier: 2,                      // Increase delay 2x each time
			Jitter:     0.2,                    // Randomize slightly to avoid thundering herd
			MaxDelay:   5 * time.Second,
		},
		MinConnectTimeout: 5 * time.Second,
	}
	// Create the Client (Non-Blocking by default)
	conn, err := grpc.NewClient("localhost:8090",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(backoffConfig),
	)
	if err != nil {
		logrus.Fatalf("Failed to create gRPC client: %v", err)
	}
	logrus.Info("Connecting to Auth Service...")
	conn.Connect() // Force connection attempt
	// Set Global State
	app.SetState(conn)
	logrus.Info("🔗 GRPC Connected to Auth Service")

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
