package servers

import (
	"net/http"

	"github.com/himanshu3889/discore-backend/base/databases"
	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/utils"
	"github.com/himanshu3889/discore-backend/configs"
	"github.com/himanshu3889/discore-backend/internal/gateway"
	authGrpc "github.com/himanshu3889/discore-backend/internal/gateway/authenticationService/grpc"
	authpb "github.com/himanshu3889/discore-backend/protos/auth"
	serverUtils "github.com/himanshu3889/discore-backend/servers/lib"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type GatewayServer struct {
	addr string
}

func NewGatewayServer(addr string) *GatewayServer {
	return &GatewayServer{
		addr: addr,
	}
}

func (s *GatewayServer) Initialize() *serverUtils.UnifiedServer {
	configs.InitializeConfigs()
	// logrus.SetFormatter(&utils.LogrusColorFormatter{})

	utils.InitSnowflake(1) // machineID = 1
	redisDatabase.InitRedis()
	database.InitPostgresDB()

	httpHandler := gateway.NewGateway().GetEngine()
	grpcSrv := grpc.NewServer()

	authpb.RegisterAuthServiceServer(grpcSrv, &authGrpc.AuthServer{})
	gatewayServer := serverUtils.NewUnifiedServer(s.addr, httpHandler, grpcSrv)
	return gatewayServer

}

func (s *GatewayServer) Run() {
	gatewayServer := s.Initialize()

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Panic recovered in gateway server: %v", r)
			// restart or handle gracefully
		}
	}()

	if err := gatewayServer.Run(); err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("Gateway server error: %s", err)
	}
}
