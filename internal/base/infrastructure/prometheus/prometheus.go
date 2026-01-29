package basePrometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Use promauto to safely handle registration
var (
	PrometheusHttpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency distribution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)

	PromotheusHttpRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "route", "status"},
	)
)

// var once sync.Once

// // Initialize the prometheus
// func InitPrometheus() {
// 	once.Do(func() {
// 		logrus.Info("Initializing prometheus")
// 		prometheus.MustRegister(PrometheusHttpDuration, PrometheusHttpRequests)
// 		logrus.Info("Prometheus initialized")
// 	})
// }
