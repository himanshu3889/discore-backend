package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/sirupsen/logrus"
)

var ESClient *elasticsearch.Client

func ConnectElasticsearch() {
	cfg := elasticsearch.Config{
		Addresses: []string{
			fmt.Sprintf("http://%s:%s",
				os.Getenv("ELASTICSEARCH_HOST"),
				os.Getenv("ELASTICSEARCH_PORT")),
		},
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    5,
		RetryBackoff:  func(i int) time.Duration { return time.Duration(i) * 100 * time.Millisecond },
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Elasticsearch client creation failed")
	}

	// Test connection
	res, err := client.Info(
		client.Info.WithContext(context.Background()),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Elasticsearch connection failed")
	}
	defer res.Body.Close()

	if res.IsError() {
		logrus.Fatalf("Elasticsearch error: %s", res.String())
	}

	ESClient = client
	logrus.Info("Elasticsearch connected successfully")
}

// Helper methods
func GetIndex(ctx context.Context, indexName string) (*esapi.Response, error) {
	return ESClient.Indices.Get([]string{indexName},
		ESClient.Indices.Get.WithContext(ctx),
	)
}

func CreateIndex(ctx context.Context, indexName string, mapping string) error {
	res, err := ESClient.Indices.Create(indexName,
		ESClient.Indices.Create.WithBody(strings.NewReader(mapping)),
		ESClient.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch error: %s", res.String())
	}
	return nil
}
