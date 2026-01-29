package coreApi

import "github.com/gin-gonic/gin"

func RegisterCoreRoutes(rg *gin.RouterGroup) {
	core := rg.Group("/core/api")
	// core routes
	registerAuthRoutes(core)
	registerServerRoutes(core)
	registerChannelRoutes(core)
	registerMemberRoutes(core)
}
