package id

import (
	"github.com/google/uuid"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// Compile-time check
var _ port.IDGenerator = (*UUIDGenerator)(nil)

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator { return &UUIDGenerator{} }

func (g *UUIDGenerator) NewNodeID() valueobject.NodeID {
	return valueobject.NodeIDFromUUID(uuid.New())
}

func (g *UUIDGenerator) NewShareID() valueobject.ShareID {
	return valueobject.ShareIDFromUUID(uuid.New())
}
