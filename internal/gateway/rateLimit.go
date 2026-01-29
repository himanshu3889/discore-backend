package gateway

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

const reqPerMinute = 60

// Middleware for rate limiting using Generic Cell Rate Algorithm (GCRA)
func (g *Gateway) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// WebSocket gets stricter connection limits
		// if c.Request.URL.Path == "/api/ws" {
		// 	g.applyRateLimit(c, "rl:ws:"+c.ClientIP(), redis_rate.PerMinute(10))
		// 	return
		// }

		// HTTP routes get standard limits
		// TODO: rate limit by login and non login
		key := c.ClientIP()
		g.applyRateLimit(c, key, redis_rate.PerMinute(reqPerMinute))
	}
}

// Apply the rate limit
func (g *Gateway) applyRateLimit(c *gin.Context, key string, limit redis_rate.Limit) {
	result, err := g.limiter.Allow(c.Request.Context(), key, limit)

	c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	c.Header("X-RateLimit-Limit", strconv.Itoa(limit.Rate))
	c.Header("X-RateLimit-Reset", strconv.Itoa(int(result.ResetAfter.Seconds())))

	if err != nil || result.Allowed <= 0 {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error":       "Rate limit exceeded",
			"Retry_After": int(result.RetryAfter.Seconds()),
		})
		return
	}

	c.Next()
}
