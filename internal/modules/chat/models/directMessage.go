package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

// Conversation represents a DM conversation between two members
type Conversation struct {
	ID        snowflake.ID `db:"id,omitempty" json:"id"`  //Snowflake ID to sort the message by timestamp
	User1ID   snowflake.ID `db:"user1_id" json:"user1ID"` // Always store the smaller ID first
	User2ID   snowflake.ID `db:"user2_id" json:"user2ID"`
	CreatedAt time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time    `db:"updated_at" json:"updatedAt"`

	User1 *User         `json:"user1"`
	User2 *User         `json:"user2"`
	MeID  *snowflake.ID `json:"meID"`
}

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
