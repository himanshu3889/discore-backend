package middlewares

import (
	"discore/configs"
	"discore/internal/base/lib/passport"
	"discore/internal/base/utils"
	"net/http"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
)

type contextKey string

const userIDKey contextKey = "userID"
const userEmailKey contextKey = "email"

func AuthMiddleware() gin.HandlerFunc {
	internalSecret := []byte(configs.Config.INTERNAL_PASSPORT_SECRET)

	return func(c *gin.Context) {
		// Get the passport from header
		passportToken := c.GetHeader(passport.HeaderName)
		if passportToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no passport"})
			return
		}

		// Verify it
		p, err := passport.VerifyPassport(passportToken, internalSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bad passport"})
			return
		}

		snowflakeUserID, err := utils.ValidSnowflakeID(p.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bad passport"})
			return
		}

		// Store in context so handlers can access it
		c.Set("passport", p)
		c.Set(userIDKey, snowflakeUserID)
		c.Set(userEmailKey, p.Email)

		c.Next()
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
