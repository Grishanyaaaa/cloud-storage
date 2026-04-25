package valueobject

import (
	"fmt"

	"github.com/google/uuid"
)

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

// ParseAuditLogID восстанавливает AuditLogID из строки с валидацией формата UUID.
func ParseAuditLogID(raw string) (AuditLogID, error) {
	if _, err := uuid.Parse(raw); err != nil {
		return "", fmt.Errorf("invalid audit log id: %w", err)
	}
	return AuditLogID(raw), nil
}
