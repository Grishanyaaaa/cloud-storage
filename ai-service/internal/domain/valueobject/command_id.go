package valueobject

import (
	"github.com/google/uuid"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
)

// CommandID identifies an ai_command record.
type CommandID struct {
	value uuid.UUID
}

// NewCommandID creates a new CommandID with a generated UUID.
func NewCommandID() CommandID {
	return CommandID{value: uuid.New()}
}

// ParseCommandID parses a string into a CommandID.
func ParseCommandID(s string) (CommandID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return CommandID{}, domainerr.ErrInvalidCommandID
	}
	return CommandID{value: id}, nil
}

// CommandIDFromUUID wraps a uuid.UUID without revalidation.
func CommandIDFromUUID(id uuid.UUID) CommandID {
	return CommandID{value: id}
}

func (c CommandID) String() string             { return c.value.String() }
func (c CommandID) Value() uuid.UUID           { return c.value }
func (c CommandID) IsZero() bool               { return c.value == uuid.UUID{} }
func (c CommandID) Equals(other CommandID) bool { return c.value == other.value }
