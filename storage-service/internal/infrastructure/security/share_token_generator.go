package security

import (
	"crypto/rand"
	"fmt"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// Compile-time check
var _ port.TokenGenerator = (*ShareTokenGenerator)(nil)

// ShareTokenGenerator produces 32-byte cryptographically random share-link tokens.
type ShareTokenGenerator struct{}

func NewShareTokenGenerator() *ShareTokenGenerator { return &ShareTokenGenerator{} }

func (g *ShareTokenGenerator) NewShareToken() (valueobject.ShareToken, error) {
	buf := make([]byte, valueobject.ShareTokenRawBytes)
	if _, err := rand.Read(buf); err != nil {
		return valueobject.ShareToken{}, fmt.Errorf("read random: %w", err)
	}
	return valueobject.ShareTokenFromBytes(buf)
}
