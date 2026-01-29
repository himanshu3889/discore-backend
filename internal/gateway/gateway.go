package gateway

import (
	redisDatabase "discore/internal/base/infrastructure/redis"
	baseMiddlewares "discore/internal/base/middlewares"
	"discore/internal/base/utils"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// TODO: USE CONFIG later; Module addresses
const (
	ChatAddr      = "http://localhost:8080"
	WebsocketAddr = "http://localhost:8080"
	CoreAddr      = "http://localhost:8080"
)

type Gateway struct {
	// Privates ; lowercased
	engine  *gin.Engine
	limiter *redis_rate.Limiter
	rdb     *redis.Client
}

// Get engine
func (g *Gateway) GetEngine() *gin.Engine {
	return g.engine
}

func NewGateway() *Gateway {
	redisDatabase.InitRedis() // Redis initialization
	redisClient := redisDatabase.RedisClient

	g := &Gateway{
		engine:  gin.New(),
		limiter: redis_rate.NewLimiter(redisClient),
		rdb:     redisClient,
	}

	g.setupRoutes()
	return g
}

func (g *Gateway) setupRoutes() {

	g.engine.Use(func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("PANIC: %v\nStack: %s", r, debug.Stack())
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	})
	// Global middleware
	// g.engine.Use(g.loggerMiddleware())
	g.engine.Use(gin.Recovery())

	// Health check
	g.engine.GET("/health", func(c *gin.Context) {
		utils.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "OK"})
	})

	// Prometheus
	g.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Private routes with rate limiting and auth
	private := g.engine.Group("")
	// Authentication
	// private.Use(g.authenticationMiddleware())  // TODO: handle authentication at this layer
	// Rate limit middleware
	private.Use(g.rateLimitMiddleware())
	// Metrics logging
	private.Use(baseMiddlewares.MetricsMiddleware())
	private.Use(baseMiddlewares.LatencyLoggerMiddleware()) // Middleware for the latency logging

	{
		private.GET("/ws", g.proxyHandler(WebsocketAddr))
		private.Any("/chat/api/*path", g.proxyHandler(ChatAddr))
		private.Any("/core/api/*path", g.proxyHandler(CoreAddr))
	}
}
