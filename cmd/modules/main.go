package main

import (
	"github.com/himanshu3889/discore-backend/configs"
	"github.com/himanshu3889/discore-backend/logger"
	"github.com/himanshu3889/discore-backend/servers"
)

func main() {
	configs.InitializeConfigs()

	// Init with specific service name
	logger.InitLogger("discore-modules")

	moduleServer := servers.NewModuleServer(":8080")
	moduleServer.Run()
}
