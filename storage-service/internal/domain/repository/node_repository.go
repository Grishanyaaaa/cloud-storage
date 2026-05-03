package repository

import (
	"context"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// NodeFilter narrows ListChildren queries.
type NodeFilter struct {
	IncludeDeleted bool
	Cursor         string // opaque cursor for pagination
	Limit          int
}

// NodeRepository defines persistence operations for Node entities.
// Implemented in infrastructure layer. Returns sentinels on known failures.
type NodeRepository interface {
	// Create persists a new node.
	// Returns domainerr.ErrNodeNameTaken on (parent_id, name) collision.
	Create(ctx context.Context, n *entity.Node) error

	// CreateTx persists a new node within a transaction.
	CreateTx(ctx context.Context, tx Transaction, n *entity.Node) error

	// GetByID retrieves a node by ID.
	// Returns domainerr.ErrNodeNotFound if not exists.
	GetByID(ctx context.Context, id valueobject.NodeID) (*entity.Node, error)

	// GetByIDForOwner is GetByID + ownership check (returns ErrNodeNotFound for foreign nodes).
	GetByIDForOwner(ctx context.Context, owner valueobject.UserID, id valueobject.NodeID) (*entity.Node, error)

	// GetRootByOwner retrieves the root node of an owner.
	// Returns domainerr.ErrUserRootNotFound if the user has no root yet.
	GetRootByOwner(ctx context.Context, owner valueobject.UserID) (*entity.Node, error)

	// ListChildren returns alive children of a folder (paginated).
	ListChildren(ctx context.Context, owner valueobject.UserID, parentID valueobject.NodeID, f NodeFilter) ([]*entity.Node, string, error)

	// ListSubtree returns nodes covered by root path up to maxDepth (root inclusive).
	// maxDepth ≤ 0 means unbounded.
	ListSubtree(ctx context.Context, owner valueobject.UserID, root *entity.Node, maxDepth int, includeDeleted bool) ([]*entity.Node, error)

	// Rename updates only the name of a node (no path changes).
	Rename(ctx context.Context, n *entity.Node) error

	// MoveTx updates parent_id, path, depth and updated_at of the moved node and rewrites
	// path/depth of every descendant in a single transaction.
	MoveTx(ctx context.Context, tx Transaction, oldPath valueobject.NodePath, n *entity.Node) error

	// SoftDeleteSubtreeTx marks `root` and every descendant as deleted at `now`.
	// Returns the number of newly-deleted rows.
	SoftDeleteSubtreeTx(ctx context.Context, tx Transaction, root *entity.Node, now time.Time) (int64, error)

	// RestoreSubtreeTx clears deleted_at on `root` and every descendant.
	RestoreSubtreeTx(ctx context.Context, tx Transaction, root *entity.Node, now time.Time) (int64, error)
}
