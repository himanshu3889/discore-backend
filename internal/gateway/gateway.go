package gateway

import (
	"net/http"
	"runtime/debug"

	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/base/middlewares"
	"github.com/himanshu3889/discore-backend/base/utils"
	authentictionApi "github.com/himanshu3889/discore-backend/internal/gateway/authenticationService/api"
	authMiddleware "github.com/himanshu3889/discore-backend/internal/gateway/authenticationService/middlewares"
	rateLimitingMiddleware "github.com/himanshu3889/discore-backend/internal/gateway/rateLimitService/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// TODO: USE CONFIG later; Module addresses
const (
	ChatAddr = "http://localhost:8080"
	CoreAddr = "http://localhost:8080"
)

// Gateway structure
type Gateway struct {
	// Privates ; lowercased
	engine  *gin.Engine
	limiter *redis_rate.Limiter
	rdb     *redis.Client
}

// Get gateway engine
func (g *Gateway) GetEngine() *gin.Engine {
	return g.engine
}

// New gateway
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

// Setup gateway routes
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
	g.engine.GET("/health-check", func(c *gin.Context) {
		utils.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "OK"})
	})

	// Prometheus
	g.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Public routes
	public := g.engine.Group("/api")
	g.addPublicMiddleware(public)
	authentictionApi.RegisterAuthRoutes(public)

	// Private routes
	private := g.engine.Group("")
	g.addPrivateMiddleware(private)

	{
		private.Any("/chat/api/*path", g.proxyHandler(ChatAddr))
		private.Any("/core/api/*path", g.proxyHandler(CoreAddr))
	}
}

// Public routes
func (g *Gateway) addPublicMiddleware(group *gin.RouterGroup) {
	group.Use(middlewares.CORSMiddleware())
	group.Use(middlewares.MetricsMiddleware()) // Captures metrics
	group.Use(middlewares.LatencyLoggerMiddleware())
	group.Use(middlewares.RequestIDMiddleware())
	group.Use(rateLimitingMiddleware.RateLimitMiddleware(g.limiter))
}

// Private routes
func (g *Gateway) addPrivateMiddleware(group *gin.RouterGroup) {
	group.Use(middlewares.CORSMiddleware())
	group.Use(middlewares.MetricsMiddleware()) // Captures metrics
	group.Use(middlewares.LatencyLoggerMiddleware())
	group.Use(authMiddleware.JwtAuthMiddleware(true))
	group.Use(middlewares.RequestIDMiddleware())
	group.Use(rateLimitingMiddleware.RateLimitMiddleware(g.limiter))
}
