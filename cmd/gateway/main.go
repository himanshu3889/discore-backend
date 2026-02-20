package main

import (
	"github.com/himanshu3889/discore-backend/configs"
	"github.com/himanshu3889/discore-backend/logger"
	"github.com/himanshu3889/discore-backend/servers"
)

func main() {
	configs.InitializeConfigs() // Load your .env first!

	// Init with specific service name
	logger.InitLogger("discore-gateway")

	gatewayServer := servers.NewGatewayServer(":8090")

	gatewayServer.Run()
}
