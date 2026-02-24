package modelsLib

import (
	"time"

	"github.com/himanshu3889/discore-backend/base/lib/appError"
	"github.com/himanshu3889/discore-backend/base/models"
)

// Pass the model in as an argument instead of a receiver
func ValidateServerInvite(invite *models.ServerInvite) *appError.Error {
	if invite.ExpiresAt != nil && time.Now().After(*invite.ExpiresAt) {
		return &appError.Error{Message: "Invite expired", Code: appError.StatusGone}
	}

	if invite.MaxUses != nil && *invite.MaxUses <= 0 {
		return &appError.Error{Message: "Invite code limit is zero or negative", Code: appError.StatusGone}
	}

	return nil
}
