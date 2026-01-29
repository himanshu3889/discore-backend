package baseMiddlewares

import (
	basePrometheus "discore/internal/base/infrastructure/prometheus"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Metric middleware - records Prometheus metrics only
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start)
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		status := strconv.Itoa(c.Writer.Status())

		// Record metrics - run async so it never blocks response
		go func() {
			basePrometheus.PrometheusHttpDuration.WithLabelValues(c.Request.Method, route, status).Observe(duration.Seconds())
			basePrometheus.PromotheusHttpRequests.WithLabelValues(c.Request.Method, route, status).Inc()
		}()
	}
}
