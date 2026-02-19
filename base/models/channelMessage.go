package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

// Message represents a message in a server channel
type ChannelMessage struct {
	ID        snowflake.ID `bson:"_id,omitempty" json:"id"` //Snowflake ID
	Content   string       `bson:"content" json:"content"`
	FileURL   *string      `bson:"file_url,omitempty" json:"fileUrl,omitempty"` // Pointer for optional field
	UserID    snowflake.ID `bson:"user_id" json:"userID"`                       // Who sent it
	ServerID  snowflake.ID `bson:"server_id" json:"serverID"`
	ChannelID snowflake.ID `bson:"channel_id" json:"channelID"` // Which channel
	Deleted   *bool        `bson:"deleted" json:"-"`
	CreatedAt time.Time    `bson:"created_at" json:"createdAt"`
	EditedAt  *time.Time   `bson:"edited_at" json:"editedAt"`
	User      *User        `json:"user"` // not in db; user send
	// Mentions:
	// ReferencedMessageID:
}
