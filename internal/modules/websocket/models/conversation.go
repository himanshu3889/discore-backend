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
}
