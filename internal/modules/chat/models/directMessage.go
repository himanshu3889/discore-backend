package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversation represents a DM conversation between two members
type Conversation struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`        //Snowflake ID to sort the message by timestamp
	MemberOneID primitive.ObjectID `bson:"memberOneID" json:"memberOneID"` // Always store the smaller ID first
	MemberTwoID primitive.ObjectID `bson:"memberTwoID" json:"memberTwoID"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
}

// DirectMessage represents a message in a DM conversation
type DirectMessage struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Content        string             `bson:"content" json:"content"`
	FileURL        *string            `bson:"fileUrl,omitempty" json:"fileUrl,omitempty"`
	MemberID       primitive.ObjectID `bson:"memberID" json:"memberID"`             // Who sent it
	ConversationID primitive.ObjectID `bson:"conversationID" json:"conversationID"` // Which DM thread
	Deleted        bool               `bson:"deleted" json:"deleted"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt" json:"updatedAt"`
}
