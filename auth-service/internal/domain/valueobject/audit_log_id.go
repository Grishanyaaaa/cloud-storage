package valueobject

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidAuditLogID is returned when an audit log ID is invalid.
var ErrInvalidAuditLogID = errors.New("invalid audit log ID")

// AuditLogID represents a unique identifier for an audit log entry.
// This is a value object that enforces the invariant of a valid UUID.
type AuditLogID struct {
	value uuid.UUID
}

// NewAuditLogID generates a new unique AuditLogID.
func NewAuditLogID() AuditLogID {
	return AuditLogID{value: uuid.New()}
}

// ParseAuditLogID parses a string into an AuditLogID.
// Returns an error if the string is not a valid UUID.
func ParseAuditLogID(s string) (AuditLogID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return AuditLogID{}, ErrInvalidAuditLogID
	}
	return AuditLogID{value: id}, nil
}

// MustParseAuditLogID parses a string into an AuditLogID.
// Panics if the string is not a valid UUID.
func MustParseAuditLogID(s string) AuditLogID {
	id, err := ParseAuditLogID(s)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the string representation of AuditLogID.
func (id AuditLogID) String() string {
	return id.value.String()
}

// Value returns the underlying UUID value.
func (id AuditLogID) Value() uuid.UUID {
	return id.value
}

// IsZero returns true if the AuditLogID is the zero value.
func (id AuditLogID) IsZero() bool {
	return id.value == uuid.UUID{}
}

// Equals checks if two AuditLogIDs are equal.
func (id AuditLogID) Equals(other AuditLogID) bool {
	return id.value == other.value
}
