package repository

import (
	"context"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// FileBlobRepository defines persistence operations for FileBlob entities.
type FileBlobRepository interface {
	// Create persists a new pending blob.
	Create(ctx context.Context, b *entity.FileBlob) error

	// CreateTx persists a new pending blob within a transaction.
	CreateTx(ctx context.Context, tx Transaction, b *entity.FileBlob) error

	// GetByNodeID returns the blob attached to a file node.
	// Returns domainerr.ErrFileBlobNotFound if not exists.
	GetByNodeID(ctx context.Context, nodeID valueobject.NodeID) (*entity.FileBlob, error)

	// Update updates an existing blob (status / size / checksum / expires_at).
	Update(ctx context.Context, b *entity.FileBlob) error

	// UpdateTx updates an existing blob within a transaction.
	UpdateTx(ctx context.Context, tx Transaction, b *entity.FileBlob) error

	// FailExpiredPending marks all pending blobs whose expires_at < now as failed.
	// limit caps the number of rows touched per call (janitor batching).
	// Returns the number of updated rows.
	FailExpiredPending(ctx context.Context, now time.Time, limit int) (int64, error)
}
