package main

import (
	"sync"

	"github.com/sirupsen/logrus"
)

func main() {
	var wg sync.WaitGroup

	// Create servers
	moduleServer := NewModuleServer(":8080")
	gatewayServer := NewGatewayServer(":8090")

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

	logrus.Info("Both servers started. Waiting for shutdown signals...")

	wg.Wait()

	logrus.Info("All servers shut down. Exiting.")
}
