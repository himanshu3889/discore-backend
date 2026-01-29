package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type Server struct {
	ID        snowflake.ID `db:"id" json:"id"`
	Name      string       `db:"name" json:"name"`
	ImageUrl  string       `db:"image_url" json:"imageUrl"`
	OwnerID   snowflake.ID `db:"owner_id" json:"-"`
	CreatedAt time.Time    `db:"created_at" json:"-"`
	UpdatedAt time.Time    `db:"updated_at" json:"-"`
	DeletedAt *time.Time   `db:"deleted_at" json:"-"`
}

type ServerInvite struct {
	Code      string       `db:"code" json:"code"` // primary key
	ServerID  snowflake.ID `db:"server_id" json:"serverID"`
	CreatedBy snowflake.ID `db:"created_by" json:"createdBy"`
	MaxUses   *int         `db:"max_uses" json:"maxUses"`     // null = unlimited
	UsedCount int          `db:"used_count" json:"usedCount"` // NOTE: race condition flag
	ExpiresAt *time.Time   `db:"expires_at" json:"expiresAt"` // null = never expires
	CreatedAt time.Time    `db:"created_at" json:"-"`
}
