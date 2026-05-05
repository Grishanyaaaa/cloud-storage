package entity

import (
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// Operation is a single planned action (delete / rename / move) over a node.
//
// Field semantics by Kind:
//
//	Delete: NodeID required.            NewName="", NewParentID=nil
//	Rename: NodeID + NewName required.  NewParentID=nil
//	Move:   NodeID + NewParentID req.   NewName=""
type Operation struct {
	kind        valueobject.OperationKind
	nodeID      valueobject.NodeID
	newName     string
	newParentID *valueobject.NodeID
}

// NewDeleteOperation constructs a delete op.
func NewDeleteOperation(node valueobject.NodeID) (Operation, error) {
	if node.IsZero() {
		return Operation{}, domainerr.ErrInvalidNodeID
	}
	return Operation{
		kind:   valueobject.OperationKindDelete,
		nodeID: node,
	}, nil
}

// NewRenameOperation constructs a rename op.
// `newName` is left as a raw string here — the actual NodeName validation
// (length, forbidden characters) is enforced by storage-service when applying.
// We only check non-emptiness to fail fast on obvious LLM hallucinations.
func NewRenameOperation(node valueobject.NodeID, newName string) (Operation, error) {
	if node.IsZero() {
		return Operation{}, domainerr.ErrInvalidNodeID
	}
	if newName == "" {
		return Operation{}, domainerr.ErrInvalidOperation
	}
	return Operation{
		kind:    valueobject.OperationKindRename,
		nodeID:  node,
		newName: newName,
	}, nil
}

// NewMoveOperation constructs a move op.
func NewMoveOperation(node valueobject.NodeID, newParent valueobject.NodeID) (Operation, error) {
	if node.IsZero() {
		return Operation{}, domainerr.ErrInvalidNodeID
	}
	if newParent.IsZero() {
		return Operation{}, domainerr.ErrInvalidNodeID
	}
	if node.Equals(newParent) {
		return Operation{}, domainerr.ErrInvalidOperation
	}
	parent := newParent
	return Operation{
		kind:        valueobject.OperationKindMove,
		nodeID:      node,
		newParentID: &parent,
	}, nil
}

// ReconstructOperation rebuilds an Operation from persisted fields without revalidation.
func ReconstructOperation(
	kind valueobject.OperationKind,
	nodeID valueobject.NodeID,
	newName string,
	newParentID *valueobject.NodeID,
) Operation {
	return Operation{
		kind:        kind,
		nodeID:      nodeID,
		newName:     newName,
		newParentID: newParentID,
	}
}

// Getters
func (o Operation) Kind() valueobject.OperationKind { return o.kind }
func (o Operation) NodeID() valueobject.NodeID      { return o.nodeID }
func (o Operation) NewName() string                 { return o.newName }
func (o Operation) NewParentID() *valueobject.NodeID {
	if o.newParentID == nil {
		return nil
	}
	cp := *o.newParentID
	return &cp
}
