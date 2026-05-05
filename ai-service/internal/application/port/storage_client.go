package port

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// TreeNode is a flattened, ai-service-internal representation of a node in the
// user's tree (returned by storage-service /tree). We only keep what's needed
// to (1) build the LLM prompt and (2) validate planned operations.
//
//	ID       — node id (UUID)
//	ParentID — parent node id (nil for root)
//	Kind     — "folder" or "file"
//	Name     — visible name
//	Path     — materialized path "/{owner}/.../{id}" (debug/log only)
//	Depth    — 1 for root, +1 per level
type TreeNode struct {
	ID       valueobject.NodeID
	ParentID *valueobject.NodeID
	Kind     string
	Name     string
	Path     string
	Depth    int
}

// StorageClient is the HTTP-level adapter to storage-service.
// All calls take the raw JWT (Bearer access token) of the caller and
// propagate it to storage-service as the Authorization header. ai-service
// itself has no service-to-service credentials.
//
// Returned errors:
//
//	domainerr.ErrStorageServiceUnavailable — transport / 5xx
//	domainerr.ErrInvalidNodeID             — 404 on a specific node
//	domainerr.ErrForbidden                 — 403
//	domainerr.ErrUnauthorized              — 401
//	other errors — wrapped storage-service domain errors (preserved Code/Message
//	if storage-service returned a structured error envelope).
type StorageClient interface {
	// GetTree returns the user's tree (their own). maxDepth ≤ 0 means default.
	// maxNodes is enforced AFTER fetching: implementations may request a deeper
	// tree and prune client-side, or (more efficiently) pass `depth` to storage.
	GetTree(ctx context.Context, jwt string, maxDepth, maxNodes int) ([]TreeNode, error)

	// DeleteNode soft-deletes a node by ID.
	DeleteNode(ctx context.Context, jwt string, nodeID valueobject.NodeID) error

	// RenameNode renames a node by ID.
	RenameNode(ctx context.Context, jwt string, nodeID valueobject.NodeID, newName string) error

	// MoveNode moves a node under newParentID.
	MoveNode(ctx context.Context, jwt string, nodeID valueobject.NodeID, newParentID valueobject.NodeID) error
}
