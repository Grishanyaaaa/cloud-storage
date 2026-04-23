package valueobject

import "github.com/google/uuid"

// AuditLogID represents a unique identifier for an audit log entry.
type AuditLogID string

// NewAuditLogID generates a new unique AuditLogID.
func NewAuditLogID() AuditLogID {
	return AuditLogID(uuid.New().String())
}

// String returns the string representation of AuditLogID.
func (id AuditLogID) String() string {
	return string(id)
}
