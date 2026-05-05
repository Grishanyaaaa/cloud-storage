package id

import (
	"github.com/google/uuid"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// Compile-time check
var _ port.IDGenerator = (*UUIDGenerator)(nil)

// UUIDGenerator implements port.IDGenerator using google/uuid v4.
type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator { return &UUIDGenerator{} }

// NewCommandID returns a fresh UUID v4 wrapped as a CommandID.
func (g *UUIDGenerator) NewCommandID() valueobject.CommandID {
	return valueobject.CommandIDFromUUID(uuid.New())
}
