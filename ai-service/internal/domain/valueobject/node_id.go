package valueobject

import (
	"github.com/google/uuid"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
)

// NodeID identifies a node (folder or file) in storage-service.
// ai-service treats it as an opaque external identifier.
type NodeID struct {
	value uuid.UUID
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
