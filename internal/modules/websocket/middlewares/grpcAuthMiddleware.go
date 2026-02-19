package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/himanshu3889/discore-backend/base/utils"
	"github.com/himanshu3889/discore-backend/internal/modules/websocket/grpcService"
	"github.com/sirupsen/logrus"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
)

type contextKey string

const userIDKey contextKey = "userID"
const userEmailKey contextKey = "email"

func GrpcAuthMiddleware() gin.HandlerFunc {

	return func(ctx *gin.Context) {
		// Get the passport from header
		authHeader := ctx.GetHeader("Authorization")

		// Extra layer specially for the websockets
		if authHeader == "" {
			cookie, err := ctx.Cookie("accessToken")
			if err == nil {
				authHeader = fmt.Sprintf("Bearer %s", cookie)
			}
		}

		if authHeader == "" {
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Missing Token")
			ctx.Abort()
			return
		}

		// Do some checkups to avoid the grpc call
		// Must start with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Invalid authorization scheme")
			ctx.Abort()
			return
		}

		accessToken := strings.TrimPrefix(authHeader, "Bearer ")
		if accessToken == "" {
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Unauthorized")
			return
		}

		resp, err := grpcService.ValidateToken(accessToken)
		if err != nil {
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Unauthorized")
			logrus.WithError(err).Error("grpc error")
			return
		}

		userID := snowflake.ID(resp.UserID)
		email := resp.Email

		// Store in context so handlers can access it
		ctx.Set(userIDKey, userID)
		ctx.Set(userEmailKey, email)

		ctx.Next()
	}
}

func GetWsContextValue[T any](ctx *gin.Context, key contextKey) (T, bool) {
	if val, exists := ctx.Get(key); exists {
		if typed, ok := val.(T); ok {
			return typed, true
		}
	}
	var zero T
	return zero, false
}

func GetWsContextUserIDEmail(ctx *gin.Context) (userID snowflake.ID, email string, ok bool) {
	userID, ok1 := GetWsContextValue[snowflake.ID](ctx, userIDKey)
	email, ok2 := GetWsContextValue[string](ctx, userEmailKey)

	if ok1 && ok2 {
		return userID, email, true
	}
	return 0, "", false
}
