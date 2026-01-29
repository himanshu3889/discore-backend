package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type ChannelType string

const (
	ChannelTypeText  ChannelType = "TEXT"
	ChannelTypeAudio ChannelType = "AUDIO"
	ChannelTypeVideo ChannelType = "VIDEO"
)

type Channel struct {
	ID        snowflake.ID `db:"id" json:"id"`
	Name      string       `db:"name" json:"name"`
	Type      ChannelType  `db:"type" json:"type"`
	CreatorID snowflake.ID `db:"creator_id" json:"creatorID"`
	ServerID  snowflake.ID `db:"server_id" json:"serverID"`
	CreatedAt time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time    `db:"updated_at" json:"updatedAt"`
	DeletedAt *time.Time   `db:"deleted_at" json:"-"`
}
