package valueobject

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidRefreshTokenID is returned when a refresh token ID is invalid.
var ErrInvalidRefreshTokenID = errors.New("invalid refresh token ID")

// RefreshTokenID represents a unique identifier for a refresh token.
// This is a value object that enforces the invariant of a valid UUID.
type RefreshTokenID struct {
	value uuid.UUID
}

// NewRefreshTokenID creates a new RefreshTokenID with a generated UUID.
func NewRefreshTokenID() RefreshTokenID {
	return RefreshTokenID{value: uuid.New()}
}

// ParseRefreshTokenID parses a string into a RefreshTokenID.
// Returns an error if the string is not a valid UUID.
func ParseRefreshTokenID(s string) (RefreshTokenID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return RefreshTokenID{}, ErrInvalidRefreshTokenID
	}
	return RefreshTokenID{value: id}, nil
}

// MustParseRefreshTokenID parses a string into a RefreshTokenID.
// Panics if the string is not a valid UUID.
func MustParseRefreshTokenID(s string) RefreshTokenID {
	id, err := ParseRefreshTokenID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of the RefreshTokenID.
func (id RefreshTokenID) String() string {
	return id.value.String()
}

// Value returns the underlying UUID value.
func (id RefreshTokenID) Value() uuid.UUID {
	return id.value
}

// IsZero returns true if the RefreshTokenID is the zero value.
func (id RefreshTokenID) IsZero() bool {
	return id.value == uuid.UUID{}
}

// Equals checks if two RefreshTokenIDs are equal.
func (id RefreshTokenID) Equals(other RefreshTokenID) bool {
	return id.value == other.value
}
