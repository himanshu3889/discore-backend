package logger

import (
	"io"

	"github.com/culionbear/lokirus"
	"github.com/himanshu3889/discore-backend/configs"
	"github.com/sirupsen/logrus"
)

func InitLogger(serviceName string) {

	configs.InitializeConfigs()

	// SILENCE THE CONSOLE
	logrus.SetOutput(io.Discard)

	// Configure Loki options
	opts := lokirus.NewLokiHookOptions().
		WithStaticLabels(lokirus.Labels{
			"app":          configs.Config.APP_NAME,
			"service_name": serviceName,
			"env":          configs.Config.ENV,
		}).
		// Optional: Add Basic Auth if using Grafana Cloud
		// WithBasicAuth("user_id", "api_key").
		WithLevelMap(lokirus.LevelMap{
			logrus.PanicLevel: "critical",
		})

	// Initialize the hook
	hook := lokirus.NewLokiHookWithOpts(
		configs.Config.LOKI_URL,
		opts,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel,
	)

	// CRITICAL: Check if hook is nil before using it
	if hook != nil {
		logrus.AddHook(hook)
	}

	// Attach directly to the GLOBAL logrus instance
	// logrus.AddHook(hook)

	// Optional: Set global formatting (JSON is best for Loki)
	logrus.SetFormatter(&logrus.JSONFormatter{})
}
