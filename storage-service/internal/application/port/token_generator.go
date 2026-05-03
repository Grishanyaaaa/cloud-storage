package port

import (
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// TokenGenerator produces fresh share-link tokens.
// Implementation must use crypto/rand.
type TokenGenerator interface {
	NewShareToken() (valueobject.ShareToken, error)
}
