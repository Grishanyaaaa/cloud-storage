package entity

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// Action represents the type of action being audited.
type Action string

const (
	ActionRegister   Action = "register"
	ActionLogin      Action = "login"
	ActionLogout     Action = "logout"
	ActionRefresh    Action = "refresh_token"
	ActionGetProfile Action = "get_profile"
	ActionDeactivate Action = "deactivate"
)

// AuditLog represents an audit log entry for tracking user actions.
type AuditLog struct {
	id        valueobject.AuditLogID
	userID    valueobject.UserID
	action    Action
	ipAddress string
	userAgent string
	createdAt time.Time
}

// NewAuditLog creates a new AuditLog entity.
// Time is injected for testability — no hidden time.Now() dependency.
func NewAuditLog(
	id valueobject.AuditLogID,
	userID valueobject.UserID,
	action Action,
	ipAddress string,
	userAgent string,
	now time.Time,
) *AuditLog {
	return &AuditLog{
		id:        id,
		userID:    userID,
		action:    action,
		ipAddress: ipAddress,
		userAgent: userAgent,
		createdAt: now,
	}
}

// ReconstructAuditLog reconstructs an AuditLog from persistence.
// Used when loading from database — no validation, trusts the data source.
func ReconstructAuditLog(
	id valueobject.AuditLogID,
	userID valueobject.UserID,
	action Action,
	ipAddress string,
	userAgent string,
	createdAt time.Time,
) *AuditLog {
	return &AuditLog{
		id:        id,
		userID:    userID,
		action:    action,
		ipAddress: ipAddress,
		userAgent: userAgent,
		createdAt: createdAt,
	}
}

// ID returns the audit log's unique identifier.
func (a *AuditLog) ID() valueobject.AuditLogID {
	return a.id
}

// UserID returns the user ID associated with this audit log.
func (a *AuditLog) UserID() valueobject.UserID {
	return a.userID
}

// Action returns the action type.
func (a *AuditLog) Action() Action {
	return a.action
}

// IPAddress returns the IP address from which the action was performed.
func (a *AuditLog) IPAddress() string {
	return a.ipAddress
}

// UserAgent returns the user agent string.
func (a *AuditLog) UserAgent() string {
	return a.userAgent
}

// CreatedAt returns the timestamp when the action was performed.
func (a *AuditLog) CreatedAt() time.Time {
	return a.createdAt
}
