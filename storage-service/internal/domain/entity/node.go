package entity

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// Node is a folder or file inside a user's hierarchy.
//
// Nodes form a tree per owner; each node stores a materialized `path` that
// contains every ancestor's NodeID separated by '/'. The owner's root node
// has parentID = nil and path = "/{ownerID}".
type Node struct {
	id        valueobject.NodeID
	ownerID   valueobject.UserID
	parentID  *valueobject.NodeID
	kind      valueobject.NodeKind
	name      valueobject.NodeName
	path      valueobject.NodePath
	depth     int
	createdAt time.Time
	updatedAt time.Time
	deletedAt *time.Time
}

// NewRootNode creates a fresh, alive root folder for owner.
func NewRootNode(id valueobject.NodeID, owner valueobject.UserID, name valueobject.NodeName, now time.Time) *Node {
	return &Node{
		id:        id,
		ownerID:   owner,
		parentID:  nil,
		kind:      valueobject.KindFolder,
		name:      name,
		path:      valueobject.NewRootPath(owner),
		depth:     1,
		createdAt: now,
		updatedAt: now,
		deletedAt: nil,
	}
}

// NewChildNode creates a fresh, alive node under parent.
// Caller is responsible for asserting parent is a folder and not deleted.
func NewChildNode(
	id valueobject.NodeID,
	owner valueobject.UserID,
	parent *Node,
	kind valueobject.NodeKind,
	name valueobject.NodeName,
	now time.Time,
) (*Node, error) {
	if !parent.IsFolder() {
		return nil, domainerr.ErrNodeKindMismatch
	}
	if parent.IsDeleted() {
		return nil, domainerr.ErrNodeAlreadyDeleted
	}
	if !parent.OwnerID().Equals(owner) {
		return nil, domainerr.ErrForbidden
	}
	childPath, err := valueobject.AppendChild(parent.Path(), id)
	if err != nil {
		return nil, err
	}
	parentID := parent.ID()
	return &Node{
		id:        id,
		ownerID:   owner,
		parentID:  &parentID,
		kind:      kind,
		name:      name,
		path:      childPath,
		depth:     parent.Depth() + 1,
		createdAt: now,
		updatedAt: now,
		deletedAt: nil,
	}, nil
}

// ReconstructNode reconstructs a Node from persistence.
// No validation — trusts the data source.
func ReconstructNode(
	id valueobject.NodeID,
	owner valueobject.UserID,
	parentID *valueobject.NodeID,
	kind valueobject.NodeKind,
	name valueobject.NodeName,
	path valueobject.NodePath,
	depth int,
	createdAt time.Time,
	updatedAt time.Time,
	deletedAt *time.Time,
) *Node {
	return &Node{
		id:        id,
		ownerID:   owner,
		parentID:  parentID,
		kind:      kind,
		name:      name,
		path:      path,
		depth:     depth,
		createdAt: createdAt,
		updatedAt: updatedAt,
		deletedAt: deletedAt,
	}
}

// Getters
func (n *Node) ID() valueobject.NodeID         { return n.id }
func (n *Node) OwnerID() valueobject.UserID    { return n.ownerID }
func (n *Node) ParentID() *valueobject.NodeID  { return n.parentID }
func (n *Node) Kind() valueobject.NodeKind     { return n.kind }
func (n *Node) Name() valueobject.NodeName     { return n.name }
func (n *Node) Path() valueobject.NodePath     { return n.path }
func (n *Node) Depth() int                     { return n.depth }
func (n *Node) CreatedAt() time.Time           { return n.createdAt }
func (n *Node) UpdatedAt() time.Time           { return n.updatedAt }
func (n *Node) DeletedAt() *time.Time          { return n.deletedAt }
func (n *Node) IsFolder() bool                 { return n.kind.IsFolder() }
func (n *Node) IsFile() bool                   { return n.kind.IsFile() }
func (n *Node) IsRoot() bool                   { return n.parentID == nil }
func (n *Node) IsDeleted() bool                { return n.deletedAt != nil }

// Rename changes the visible name. Root cannot be renamed.
func (n *Node) Rename(newName valueobject.NodeName, now time.Time) error {
	if n.IsRoot() {
		return domainerr.ErrRootImmutable
	}
	if n.IsDeleted() {
		return domainerr.ErrNodeAlreadyDeleted
	}
	n.name = newName
	n.updatedAt = now
	return nil
}

// MoveTo updates parent / path / depth. Root cannot be moved.
// Caller is responsible for verifying parent is owned, alive, a folder, and
// not a descendant of the receiver (anti-cycle check).
func (n *Node) MoveTo(newParent *Node, now time.Time) error {
	if n.IsRoot() {
		return domainerr.ErrRootImmutable
	}
	if n.IsDeleted() {
		return domainerr.ErrNodeAlreadyDeleted
	}
	if !newParent.IsFolder() {
		return domainerr.ErrNodeKindMismatch
	}
	if newParent.IsDeleted() {
		return domainerr.ErrNodeAlreadyDeleted
	}
	if !newParent.OwnerID().Equals(n.ownerID) {
		return domainerr.ErrMoveAcrossOwners
	}
	if n.id.Equals(newParent.id) || n.path.IsAncestorOf(newParent.path) {
		return domainerr.ErrMoveIntoSelf
	}
	newPath, err := valueobject.AppendChild(newParent.Path(), n.id)
	if err != nil {
		return err
	}
	parentID := newParent.ID()
	n.parentID = &parentID
	n.path = newPath
	n.depth = newParent.Depth() + 1
	n.updatedAt = now
	return nil
}

// SoftDelete marks the node as deleted. Root cannot be soft-deleted.
func (n *Node) SoftDelete(now time.Time) error {
	if n.IsRoot() {
		return domainerr.ErrRootImmutable
	}
	if n.IsDeleted() {
		return domainerr.ErrNodeAlreadyDeleted
	}
	n.deletedAt = &now
	n.updatedAt = now
	return nil
}

// Restore removes the deletion mark.
func (n *Node) Restore(now time.Time) error {
	if !n.IsDeleted() {
		return nil
	}
	n.deletedAt = nil
	n.updatedAt = now
	return nil
}
