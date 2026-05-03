package repository

import (
	"context"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// ShareRepository defines persistence operations for share-link entities.
type ShareRepository interface {
	// Create persists a new share.
	Create(ctx context.Context, s *entity.Share) error

	// GetByID retrieves a share owned by ownerID.
	// Returns domainerr.ErrShareNotFound if not exists or not owned by ownerID.
	GetByID(ctx context.Context, ownerID valueobject.UserID, id valueobject.ShareID) (*entity.Share, error)

	// GetActiveByTokenHash retrieves an active share by its sha256(token) hash.
	// Returns domainerr.ErrShareNotFound when no row matches, ErrShareRevoked / ErrShareExpired
	// when found but not active.
	GetActiveByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*entity.Share, error)

	// ListByNode lists shares attached to a single node.
	// includeRevoked toggles whether revoked rows are returned.
	ListByNode(ctx context.Context, ownerID valueobject.UserID, nodeID valueobject.NodeID, includeRevoked bool) ([]*entity.Share, error)

	// RevokeByID marks a share owned by ownerID as revoked.
	// Returns domainerr.ErrShareNotFound if not exists.
	RevokeByID(ctx context.Context, ownerID valueobject.UserID, id valueobject.ShareID, now time.Time) error

	// RevokeSubtreeTx revokes every alive share whose node is inside (or equal to) `root`'s subtree.
	// Returns the number of newly-revoked rows.
	RevokeSubtreeTx(ctx context.Context, tx Transaction, root *entity.Node, now time.Time) (int64, error)

	// ExpireDue marks shares with expires_at ≤ now as revoked (idempotent for already-revoked rows).
	// limit caps rows touched per call. Returns the number of newly-revoked rows.
	ExpireDue(ctx context.Context, now time.Time, limit int) (int64, error)
}
