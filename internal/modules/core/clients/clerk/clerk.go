package clerkClient

import (
	"discore/configs"
	"sync"

	"github.com/clerkinc/clerk-sdk-go/clerk"
	"github.com/sirupsen/logrus"
)

var (
	ClerkClient clerk.Client
	once        sync.Once
)

// Initialize clerk client authentication
func InitializeClerk() {
	once.Do(func() {
		var err error
		// Now this assignment works correctly
		ClerkClient, err = clerk.NewClient(configs.Config.CLERK_SECRET_KEY)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to connect to clerk")
		}
	})
}
