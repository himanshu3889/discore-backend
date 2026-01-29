package gateway

import (
	"discore/internal/modules/core/middlewares"

	"github.com/gin-gonic/gin"
)

func (g *Gateway) authenticationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		middlewares.JwtAuthMiddleware()(c)
	}
}
