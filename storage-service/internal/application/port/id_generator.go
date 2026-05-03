package port

import (
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// IDGenerator produces fresh identifiers for the use case layer.
// Abstracted away so tests can supply deterministic IDs.
type IDGenerator interface {
	NewNodeID() valueobject.NodeID
	NewShareID() valueobject.ShareID
}
