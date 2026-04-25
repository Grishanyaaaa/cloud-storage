package valueobject

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidUserID is returned when a user ID is invalid.
var ErrInvalidUserID = errors.New("invalid user ID")

// UserID represents a unique identifier for a user.
// This is a value object that enforces the invariant of a valid UUID.
type UserID struct {
	value uuid.UUID
}

// NewUserID creates a new UserID with a generated UUID.
func NewUserID() UserID {
	return UserID{value: uuid.New()}
}

// ParseUserID parses a string into a UserID.
// Returns an error if the string is not a valid UUID.
func ParseUserID(s string) (UserID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return UserID{}, ErrInvalidUserID
	}
	return UserID{value: id}, nil
}

// String returns the string representation of the UserID.
func (u UserID) String() string {
	return u.value.String()
}

// Value returns the underlying UUID value.
func (u UserID) Value() uuid.UUID {
	return u.value
}

// IsZero returns true if the UserID is the zero value.
func (u UserID) IsZero() bool {
	return u.value == uuid.UUID{}
}

// Equals checks if two UserIDs are equal.
func (u UserID) Equals(other UserID) bool {
	return u.value == other.value
}
