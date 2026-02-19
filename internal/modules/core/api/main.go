package coreApi

import (
	"github.com/himanshu3889/discore-backend/base/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterCoreRoutes(rg *gin.RouterGroup) {
	core := rg.Group("/core/api", middlewares.PassportAuthMiddleware())
	// core routes
	registerServerRoutes(core)
	registerChannelRoutes(core)
	registerMemberRoutes(core)
}
