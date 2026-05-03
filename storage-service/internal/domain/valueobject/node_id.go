package valueobject

import (
	"github.com/google/uuid"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

// NodeID identifies a node (folder or file) in the storage hierarchy.
type NodeID struct {
	value uuid.UUID
}

// NewNodeID creates a new NodeID with a generated UUID.
func NewNodeID() NodeID {
	return NodeID{value: uuid.New()}
}

// ParseNodeID parses a string into a NodeID.
func ParseNodeID(s string) (NodeID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return NodeID{}, domainerr.ErrInvalidNodeID
	}
	return NodeID{value: id}, nil
}

// NodeIDFromUUID wraps a uuid.UUID as a NodeID without revalidation.
func NodeIDFromUUID(id uuid.UUID) NodeID {
	return NodeID{value: id}
}

func (n NodeID) String() string         { return n.value.String() }
func (n NodeID) Value() uuid.UUID       { return n.value }
func (n NodeID) IsZero() bool           { return n.value == uuid.UUID{} }
func (n NodeID) Equals(other NodeID) bool { return n.value == other.value }
