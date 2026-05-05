package valueobject

import "github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"

// OperationKind enumerates the operations the LLM is allowed to plan.
// Only three are supported in the MVP: delete / rename / move.
type OperationKind struct {
	value string
}

const (
	opKindDelete = "delete"
	opKindRename = "rename"
	opKindMove   = "move"
)

var (
	OperationKindDelete = OperationKind{value: opKindDelete}
	OperationKindRename = OperationKind{value: opKindRename}
	OperationKindMove   = OperationKind{value: opKindMove}
)

// ParseOperationKind validates a string against the allowed enum.
func ParseOperationKind(s string) (OperationKind, error) {
	switch s {
	case opKindDelete:
		return OperationKindDelete, nil
	case opKindRename:
		return OperationKindRename, nil
	case opKindMove:
		return OperationKindMove, nil
	default:
		return OperationKind{}, domainerr.ErrInvalidOperationKind
	}
}

func (k OperationKind) String() string { return k.value }
func (k OperationKind) IsZero() bool   { return k.value == "" }

func (k OperationKind) IsDelete() bool { return k.value == opKindDelete }
func (k OperationKind) IsRename() bool { return k.value == opKindRename }
func (k OperationKind) IsMove() bool   { return k.value == opKindMove }

func (k OperationKind) Equals(other OperationKind) bool { return k.value == other.value }
