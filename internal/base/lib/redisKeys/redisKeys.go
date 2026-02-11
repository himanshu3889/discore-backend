package rediskeys

import (
	"fmt"

	"github.com/bwmarrin/snowflake"
)

// User
type userKeys struct{}

func (k userKeys) Info(id snowflake.ID) string {
	return fmt.Sprintf("discore:user:%d:info", id)
}

// Server
type serverKeys struct{}

func (k serverKeys) Info(id snowflake.ID) string {
	return fmt.Sprintf("discore:server:%d:info", id)
}

// Channel
type channelKeys struct{}

func (k channelKeys) Info(id snowflake.ID) string {
	return fmt.Sprintf("discore:channel:%d:info", id)
}

// Server Invite
type serverInviteKeys struct{}

func (k serverInviteKeys) Info(code string) string {
	return fmt.Sprintf("discore:server_invite:%s:info", code)
}

// Usage
var Keys = struct {
	User         userKeys
	Server       serverKeys
	Channel      channelKeys
	ServerInvite serverInviteKeys
}{}
