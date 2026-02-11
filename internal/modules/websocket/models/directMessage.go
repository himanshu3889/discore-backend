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
	UserID         snowflake.ID `bson:"user_id" json:"userID"`                 // Who sent it
	ConversationID snowflake.ID `bson:"conversation_id" json:"conversationID"` // Which DM thread
	Deleted        *bool        `bson:"deleted" json:"-"`
	CreatedAt      time.Time    `bson:"created_at" json:"createdAt"`
	UpdatedAt      *time.Time   `bson:"updated_at" json:"updatedAt"`
	EditedAt       *time.Time   `bson:"edited_at" json:"editedAt"`
}
