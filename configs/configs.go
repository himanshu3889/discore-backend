package configs

import (
	"sync"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

// Config mirrors every env-var exactly; all public. so if any error can check using os.GetEnv()
type config struct {
	// App
	APP_PORT   int
	LOG_LEVEL  string
	MACHINE_ID int

	// Authentication
	JWT_SECRET               string
	INTERNAL_PASSPORT_SECRET string
	GATEWAY_SECRET           string

	// Clerk
	CLERK_SECRET_KEY string

	// PostgreSQL
	POSTGRES_HOST     string
	POSTGRES_PORT     string
	POSTGRES_USER     string
	POSTGRES_PASSWORD string
	POSTGRES_DB       string

	// MongoDB
	MONGODB_ROOT_PASSWORD string
	MONGODB_DATABASE      string
	MONGODB_USERNAME      string
	MONGODB_PASSWORD      string
	MONGODB_HOST          string

	// Redis (Base)
	REDIS_HOST     string
	REDIS_PORT     string
	REDIS_PASSWORD string

	// Redis Modules
	REDIS_BLOOM_ENABLED bool
	REDIS_CELL_ENABLED  bool

	// Kafka
	KAFKA_HOST    string
	KAFKA_PORT    string
	KAFKA_BROKERS string

	// Elasticsearch
	ELASTICSEARCH_HOST     string
	ELASTICSEARCH_PORT     string
	ELASTICSEARCH_USERNAME string
	ELASTICSEARCH_PASSWORD string

	// Rate Limiting
	RATE_LIMIT_PER_MINUTE int
}

var Config *config
var once sync.Once

func InitializeConfigs() {
	once.Do(func() {

		err := godotenv.Load()
		if err != nil {
			err = godotenv.Load("../.env") // up one level
		}

		if err != nil {
			logrus.Error("Unable to initialize configs. No .env file found!")
		}

		Config = &config{}
		if err := envconfig.Process("", Config); err != nil {
			panic("config: " + err.Error())
		}
	})
}
