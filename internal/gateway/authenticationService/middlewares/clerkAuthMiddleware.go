package middlewares

import (
	"discore/internal/base/utils"
	clerkClient "discore/internal/modules/core/clients/clerk"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func ClerkRequestMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			utils.RespondWithError(ctx, http.StatusBadRequest, "Missing Token")
			ctx.Abort()
			return
		}

		// Must start with "Clerk "
		if !strings.HasPrefix(authHeader, "Clerk ") {
			logrus.Info(authHeader)
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Invalid authorization scheme")
			ctx.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Clerk ")

		if token == "" {
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Verify token
		_, err := clerkClient.ClerkClient.VerifyToken(token)
		if err != nil {
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Unauthorized")
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
