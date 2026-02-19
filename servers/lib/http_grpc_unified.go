package serverUtils

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/utils"
	"github.com/himanshu3889/discore-backend/configs"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
)

type UnifiedServer struct {
	addr       string
	httpServer *http.Server
	grpcServer *grpc.Server
	listener   net.Listener
	mux        cmux.CMux
}

func NewUnifiedServer(addr string, httpHandler http.Handler, grpcSrv *grpc.Server) *UnifiedServer {
	return &UnifiedServer{
		addr: addr,
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      httpHandler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		grpcServer: grpcSrv,
	}
}

func (s *UnifiedServer) Run() error {
	configs.InitializeConfigs()
	logrus.SetFormatter(&utils.LogrusColorFormatter{})

	redisDatabase.InitRedis()

	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.listener = lis

	m := cmux.New(lis)
	s.mux = m

	grpcListner := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpListner := m.Match(cmux.Any())

	// --- Start gRPC ---
	go func() {
		logrus.Infof("gRPC running on %s", s.addr)
		if err := s.grpcServer.Serve(grpcListner); err != nil {
			logrus.Errorf("gRPC failed: %v", err)
		}
	}()

	// --- Start HTTP ---
	go func() {
		logrus.Infof("HTTP running on %s", s.addr)
		if err := s.httpServer.Serve(httpListner); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("HTTP failed: %v", err)
		}
	}()

	// --- Start multiplexer ---
	go func() {
		logrus.Infof("Unified server listening on %s", s.addr)
		if err := m.Serve(); err != nil {
			logrus.Errorf("cmux failed: %v", err)
		}
	}()

	// --- Wait for shutdown signal ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	signal.Stop(quit)

	logrus.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.Shutdown(ctx)

	logrus.Info("Unified server stopped gracefully")

	return nil
}

func (s *UnifiedServer) Shutdown(ctx context.Context) {
	logrus.Info("Shutting down unified server...")

	s.grpcServer.GracefulStop()
	s.httpServer.Shutdown(ctx)

	if s.listener != nil {
		s.listener.Close()
	}
}
