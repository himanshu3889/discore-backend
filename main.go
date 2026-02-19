package main

import (
	"sync"

	"github.com/himanshu3889/discore-backend/servers"
	"github.com/sirupsen/logrus"
)

func main() {
	var wg sync.WaitGroup

	// Create servers
	moduleServer := servers.NewModuleServer(":8080")

	gatewayServer := servers.NewGatewayServer(":8090")

	// Run both servers and wait for them
	wg.Add(2)

	go func() {
		defer wg.Done()
		moduleServer.Run()
	}()

	go func() {
		defer wg.Done()
		gatewayServer.Run()
	}()

	logrus.Info("All servers started. Waiting for shutdown signals...")

	wg.Wait()

	logrus.Info("All servers shut down. Exiting.")
}
