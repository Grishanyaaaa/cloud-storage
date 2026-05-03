package valueobject

import (
	"github.com/google/uuid"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

// ShareID identifies a single share-link.
type ShareID struct {
	value uuid.UUID
}

// NewShareID creates a new ShareID with a generated UUID.
func NewShareID() ShareID {
	return ShareID{value: uuid.New()}
}

// ParseShareID parses a string into a ShareID.
func ParseShareID(s string) (ShareID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return ShareID{}, domainerr.ErrInvalidShareID
	}
	return ShareID{value: id}, nil
}

// ShareIDFromUUID wraps a uuid.UUID as a ShareID without revalidation.
func ShareIDFromUUID(id uuid.UUID) ShareID {
	return ShareID{value: id}
}

func (s ShareID) String() string          { return s.value.String() }
func (s ShareID) Value() uuid.UUID        { return s.value }
func (s ShareID) IsZero() bool            { return s.value == uuid.UUID{} }
func (s ShareID) Equals(other ShareID) bool { return s.value == other.value }
