package rediskeys

import (
	"fmt"

	"github.com/bwmarrin/snowflake"
)

// User
type userKeys struct{}

func (k userKeys) Info(id snowflake.ID) (string, string) {
	return fmt.Sprintf("discore:user:%d:info", id), "user:id:info" // cacheKey, "entity:operation"
}

// Server
type serverKeys struct{}

func (k serverKeys) Info(id snowflake.ID) (string, string) {
	return fmt.Sprintf("discore:server:%d:info", id), "server:id:info"
}

// Channel
type channelKeys struct{}

func (k channelKeys) Info(id snowflake.ID) (string, string) {
	return fmt.Sprintf("discore:channel:%d:info", id), "channel:id:info"
}

// Server Invite
type serverInviteKeys struct{}

func (k serverInviteKeys) Info(code string) (string, string) {
	return fmt.Sprintf("discore:server_invite:%s:info", code), "server_invite:code:info"
}

func (k serverInviteKeys) UsedCount(code string) (string, string) {
	return fmt.Sprintf("discore:server_invite:%s:used_count", code), "server_invite:code:used_count"
}

func (k serverInviteKeys) UseCountLuaScript() string {
	return "server_invite:used_count:lua_script"
}

// Usage
var Keys = struct {
	User         userKeys
	Server       serverKeys
	Channel      channelKeys
	ServerInvite serverInviteKeys
}{}
