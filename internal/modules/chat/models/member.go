package models

import (
	"time"

	"github.com/bwmarrin/snowflake"
)

type MemberRole string

const (
	MemberRoleADMIN     MemberRole = "ADMIN"
	MemberRoleMODERATOR MemberRole = "MODERATOR"
	MemberRoleGUEST     MemberRole = "GUEST"
)

// Server member
type Member struct {
	ID        snowflake.ID `db:"id" json:"id"`
	Role      MemberRole   `db:"role" json:"role"`
	UserID    snowflake.ID `db:"user_id" json:"userID"`
	ServerID  snowflake.ID `db:"server_id" json:"serverID"`
	CreatedAt time.Time    `db:"created_at" json:"-"`
	UpdatedAt time.Time    `db:"updated_at" json:"-"`
	// could use the invite code joined; null no need foreign relation
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
	User      *User      `json:"user"` // not in db; used in join
}
