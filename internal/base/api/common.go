package baseApi

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func registerCommonRoutes(rg *gin.RouterGroup) {
	rg.GET("/health-check", healthCheck)
	rg.GET("/ready", ready)
	rg.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

func healthCheck(c *gin.Context) {
	c.String(200, "Service OK")
}

func ready(c *gin.Context) {
	c.String(200, "Ready")
}
