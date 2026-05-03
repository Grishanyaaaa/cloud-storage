package entity

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// UserRoot links a user (auth-service identity) to their root folder node.
// One row per user. Lazily created on first access.
type UserRoot struct {
	userID    valueobject.UserID
	rootID    valueobject.NodeID
	createdAt time.Time
}

// NewUserRoot creates a fresh UserRoot binding.
func NewUserRoot(userID valueobject.UserID, rootID valueobject.NodeID, now time.Time) *UserRoot {
	return &UserRoot{
		userID:    userID,
		rootID:    rootID,
		createdAt: now,
	}
}

// ReconstructUserRoot reconstructs a UserRoot from persistence.
func ReconstructUserRoot(userID valueobject.UserID, rootID valueobject.NodeID, createdAt time.Time) *UserRoot {
	return &UserRoot{
		userID:    userID,
		rootID:    rootID,
		createdAt: createdAt,
	}
}

func (u *UserRoot) UserID() valueobject.UserID { return u.userID }
func (u *UserRoot) RootID() valueobject.NodeID { return u.rootID }
func (u *UserRoot) CreatedAt() time.Time       { return u.createdAt }
