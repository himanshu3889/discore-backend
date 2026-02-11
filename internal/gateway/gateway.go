package gateway

import (
	redisDatabase "discore/internal/base/infrastructure/redis"
	baseMiddlewares "discore/internal/base/middlewares"
	"discore/internal/base/utils"
	authentictionApi "discore/internal/gateway/authenticationService/api"
	authMiddleware "discore/internal/gateway/authenticationService/middlewares"
	rateLimitingMiddleware "discore/internal/gateway/rateLimitService/middlewares"
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
		private.GET("/ws", g.proxyHandler(WebsocketAddr))
		private.Any("/chat/api/*path", g.proxyHandler(ChatAddr))
		private.Any("/core/api/*path", g.proxyHandler(CoreAddr))
	}
}

// Public routes
func (g *Gateway) addPublicMiddleware(group *gin.RouterGroup) {
	group.Use(baseMiddlewares.CORSMiddleware())
	group.Use(baseMiddlewares.RequestIDMiddleware())
	group.Use(rateLimitingMiddleware.RateLimitMiddleware(g.limiter))
	group.Use(baseMiddlewares.LatencyLoggerMiddleware())
}

// Private routes
func (g *Gateway) addPrivateMiddleware(group *gin.RouterGroup) {
	group.Use(baseMiddlewares.CORSMiddleware())
	group.Use(baseMiddlewares.RequestIDMiddleware())
	group.Use(authMiddleware.JwtAuthMiddleware(true))
	group.Use(rateLimitingMiddleware.RateLimitMiddleware(g.limiter))
	group.Use(baseMiddlewares.LatencyLoggerMiddleware())
}
