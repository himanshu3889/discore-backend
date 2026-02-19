package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/himanshu3889/discore-backend/base/lib/passport"
	"github.com/himanshu3889/discore-backend/base/utils"
	"github.com/himanshu3889/discore-backend/configs"
	"github.com/himanshu3889/discore-backend/internal/gateway/authenticationService/jwtAuthentication"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "userID"
const userEmailKey contextKey = "email"

// AuthMiddleware is a function that returns another function of type gin.HandlerFunc.
// gin.HandlerFunc is a function with one parameter c *gin.Context, representing the context of the current HTTP request and response.
func JwtAuthMiddleware(externalService bool, allowedRoles ...string) gin.HandlerFunc {
	var jwtSecret = []byte(configs.Config.JWT_SECRET)
	var passportSecret = []byte(configs.Config.INTERNAL_PASSPORT_SECRET)
	return func(ctx *gin.Context) {
		// SECURITY: Strip any incoming internal headers first!
		// This prevents header injection attacks
		ctx.Request.Header.Del(passport.HeaderName)
		ctx.Request.Header.Del("X-User-ID")
		ctx.Request.Header.Del("X-User-Email")

		authHeader := ctx.GetHeader("Authorization")

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
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Invalid token")
			return
		}
		claims := token.Claims.(*jwtAuthentication.JwtClaims)
		userID := claims.UserId
		email := claims.Email
		// For use it internally
		ctx.Set(userEmailKey, email)
		ctx.Set(userIDKey, userID)

		// For external services
		if externalService {
			// Create internal passport (short lived, 5 minutes)
			p := passport.Passport{
				UserID:    userID.String(),
				Email:     email,
				Roles:     []string{"user"}, // You can extract from claims too
				IssuedAt:  time.Now().Unix(),
				ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
			}

			// Sign it
			passportToken := passport.SignPassport(p, passportSecret)

			// Add to request header
			ctx.Request.Header.Set(passport.HeaderName, passportToken)

			// Also set plain headers for convenience (optional)
			ctx.Request.Header.Set("X-User-ID", userID.String())
			ctx.Request.Header.Set("X-User-Email", email)
		}

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
