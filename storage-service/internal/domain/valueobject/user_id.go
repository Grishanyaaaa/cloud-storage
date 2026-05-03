package valueobject

import (
	"github.com/google/uuid"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

// UserID is an external identifier owned by auth-service.
// storage-service stores it but never validates ownership against a users table.
type UserID struct {
	value uuid.UUID
}

// NewUserID creates a new UserID with a generated UUID.
func NewUserID() UserID {
	return UserID{value: uuid.New()}
}

// ParseUserID parses a string into a UserID.
func ParseUserID(s string) (UserID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return UserID{}, domainerr.ErrInvalidUserID
	}
	return UserID{value: id}, nil
}

// FromUUID wraps a uuid.UUID as a UserID without revalidation
// (used by repository layer when scanning from DB).
func UserIDFromUUID(id uuid.UUID) UserID {
	return UserID{value: id}
}

func (u UserID) String() string         { return u.value.String() }
func (u UserID) Value() uuid.UUID       { return u.value }
func (u UserID) IsZero() bool           { return u.value == uuid.UUID{} }
func (u UserID) Equals(other UserID) bool { return u.value == other.value }
