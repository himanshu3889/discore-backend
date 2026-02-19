package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

// DirectMessage represents a message in a DM conversation
type DirectMessage struct {
	ID             snowflake.ID `bson:"_id,omitempty" json:"id"`
	Content        string       `bson:"content" json:"content"`
	FileURL        *string      `bson:"file_url,omitempty" json:"fileUrl,omitempty"`
	ConversationID snowflake.ID `bson:"conversation_id" json:"conversationID"` // Which DM thread
	UserID         snowflake.ID `bson:"user_id" json:"userID"`                 // Who sent it
	Deleted        *bool        `bson:"deleted" json:"deleted"`
	CreatedAt      time.Time    `bson:"created_at" json:"createdAt"`
	UpdatedAt      *time.Time   `bson:"updated_at" json:"updatedAt"`
	User           *User        `json:"user"`
}
