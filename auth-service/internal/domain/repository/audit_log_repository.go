package repository

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// AuditLogRepository defines the interface for audit log persistence.
// Append-only by design — no update or delete operations.
type AuditLogRepository interface {
	// Save persists a new audit log entry.
	Save(ctx context.Context, log *entity.AuditLog) error

	// FindByUserID retrieves audit logs for a specific user, ordered by created_at desc.
	FindByUserID(ctx context.Context, userID valueobject.UserID, limit, offset int) ([]*entity.AuditLog, error)
}
