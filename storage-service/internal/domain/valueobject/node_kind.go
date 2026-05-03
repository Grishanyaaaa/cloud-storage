package valueobject

import (
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

// NodeKind enumerates supported node kinds.
type NodeKind string

const (
	KindFolder NodeKind = "folder"
	KindFile   NodeKind = "file"
)

// ParseNodeKind parses a string into a NodeKind.
func ParseNodeKind(s string) (NodeKind, error) {
	switch NodeKind(s) {
	case KindFolder, KindFile:
		return NodeKind(s), nil
	default:
		return "", domainerr.ErrInvalidNodeKind
	}
}

func (k NodeKind) String() string  { return string(k) }
func (k NodeKind) IsFolder() bool { return k == KindFolder }
func (k NodeKind) IsFile() bool   { return k == KindFile }
