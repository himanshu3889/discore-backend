package websocketApp

import (
	"encoding/json"

	"github.com/himanshu3889/discore-backend/configs"

	"github.com/go-redis/redis_rate/v10"
	"github.com/gorilla/websocket"
)

// Rate limit response structure sent to client
type RateLimitError struct {
	Event      string `json:"event"`       // "rate_limit"
	Error      string `json:"error"`       // "Too many messages"
	RetryAfter int    `json:"retry_after"` // seconds
	Reset      int    `json:"reset"`
	Limit      int    `json:"limit"`
}

// Ratelimiting: Returns true if allowed, false if blocked
func (hub *Hub) ApplyRateLimit(client *Client) bool {
	// Use userID as key (consistent with your HTTP middleware)
	key := client.userID.String()

	limit := configs.Config.RATE_LIMIT_PER_MINUTE
	result, err := hub.limiter.Allow(hub.ctx, key, redis_rate.PerMinute(limit))
	if err != nil {
		// Fail open on Redis errors to avoid blocking all users
		return true
	}

	if result.Allowed == 0 {
		// Send rate limit message via write pump (thread-safe)
		msg := RateLimitError{
			Event:      "rate_limit",
			Error:      "Too many messages. Slow down.",
			RetryAfter: int(result.RetryAfter.Seconds()),
			Reset:      int(result.ResetAfter.Seconds()),
			Limit:      limit,
		}

		messageBytes, _ := json.Marshal(msg)
		preparedMsg, err := websocket.NewPreparedMessage(websocket.TextMessage, messageBytes)
		if err != nil {
			return true
		}

		select {
		case client.send <- preparedMsg:
			// Message queued for write pump
		default:
			// Send buffer full, close connection
			return true

		}
		return false
	}

	return true
}
