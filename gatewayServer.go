package main

import (
	"context"
	"discore/configs"
	redisDatabase "discore/internal/base/infrastructure/redis"
	"discore/internal/base/utils"
	"discore/internal/gateway"
	gatewayAuthDatabase "discore/internal/gateway/authenticationService/database"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

type GatewayServer struct {
	server  *http.Server
	addr    string
	gateway *gateway.Gateway
}

func NewGatewayServer(addr string) *GatewayServer {
	return &GatewayServer{
		addr: addr,
	}
}

func (s *GatewayServer) Initialize() {
	configs.InitializeConfigs()
	logrus.SetFormatter(&utils.LogrusColorFormatter{})

	redisDatabase.InitRedis()

	s.gateway = gateway.NewGateway()

	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      s.gateway.GetEngine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	gatewayAuthDatabase.InitPostgresDB()
	// Note: Redis is initialized twice in your original code,
	// but once is enough if they share the same instance
}

func (s *GatewayServer) Start() error {
	logrus.Infof("Starting gateway server on %s", s.addr)
	return s.server.ListenAndServe()
}

func (s *GatewayServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *GatewayServer) Run() {
	s.Initialize()

	// Start server in goroutine
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

	logrus.Info("Shutting down gateway server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		logrus.Fatalf("Gateway server forced to shutdown: %v", err)
	}

	logrus.Info("Gateway server exited")
}
