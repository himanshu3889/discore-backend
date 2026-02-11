package coreApi

import (
	"discore/internal/modules/core/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterCoreRoutes(rg *gin.RouterGroup) {
	core := rg.Group("/core/api", middlewares.AuthMiddleware())
	// core routes
	registerServerRoutes(core)
	registerChannelRoutes(core)
	registerMemberRoutes(core)
}
