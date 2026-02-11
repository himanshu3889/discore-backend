package middlewares

import (
	"discore/configs"
	"discore/internal/gateway/authenticationService/middlewares"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

// Middleware for rate limiting using Generic Cell Rate Algorithm (GCRA)
func RateLimitMiddleware(limiter *redis_rate.Limiter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		key := ctx.ClientIP()
		userID, _, isOk := middlewares.GetContextUserIDEmail(ctx)
		if isOk {
			key = userID.String()
		}
		applyRateLimit(ctx, limiter, key, redis_rate.PerMinute(configs.Config.RATE_LIMIT_PER_MINUTE))
	}
}

// Apply the rate limit
func applyRateLimit(ctx *gin.Context, limiter *redis_rate.Limiter, key string, limit redis_rate.Limit) {
	result, err := limiter.Allow(ctx.Request.Context(), key, limit)

	ctx.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	ctx.Header("X-RateLimit-Limit", strconv.Itoa(limit.Rate))
	ctx.Header("X-RateLimit-Reset", strconv.Itoa(int(result.ResetAfter.Seconds())))

	if err != nil || result.Allowed <= 0 {
		ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error":       "Rate limit exceeded",
			"Retry_After": int(result.RetryAfter.Seconds()),
		})
		return
	}

	ctx.Next()
}
