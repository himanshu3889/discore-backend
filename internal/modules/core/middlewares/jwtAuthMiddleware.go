package middlewares

import (
	"discore/internal/base/utils"
	"discore/internal/modules/core/services/authetication/jwtAuthentication"
	"fmt"
	"net/http"
	"strings"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "userID"
const userEmailKey contextKey = "userEmail"

// AuthMiddleware is a function that returns another function of type gin.HandlerFunc.
// gin.HandlerFunc is a function with one parameter c *gin.Context, representing the context of the current HTTP request and response.
func JwtAuthMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")

		// Extra layer specially for the websockets
		//TODO: (wrong method) If authorization not provide check for the cookie
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

		token, err := jwt.ParseWithClaims(accessToken, &jwtAuthentication.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtAuthentication.JwtSecret, nil
		})
		if err != nil || !token.Valid {
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Invalid token")
			return
		}
		claims := token.Claims.(*jwtAuthentication.JwtClaims)
		ctx.Set(userEmailKey, claims.Email)
		ctx.Set(userIDKey, claims.UserId)

		ctx.Next()
	}
}

func GetContextValue[T any](ctx *gin.Context, key contextKey) (T, bool) {
	if val, exists := ctx.Get(key); exists {
		if typed, ok := val.(T); ok {
			return typed, true
		}
	}
	var zero T
	return zero, false
}

func GetContextUserIDEmail(ctx *gin.Context) (userID snowflake.ID, email string, ok bool) {
	userID, ok1 := GetContextValue[snowflake.ID](ctx, userIDKey)
	email, ok2 := GetContextValue[string](ctx, userEmailKey)

	if ok1 && ok2 {
		return userID, email, true
	}
	return 0, "", false
}
