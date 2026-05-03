package valueobject

import (
	"strings"
	"unicode/utf8"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

const (
	maxNodeNameRunes = 255
)

// NodeName is a validated node name (folder or file).
// Invariants:
//   - non-empty, ≤ 255 runes;
//   - no path separators ('/'), no NUL bytes;
//   - no leading/trailing whitespace;
//   - not "." or "..".
type NodeName struct {
	value string
}

// NewNodeName validates the supplied raw name and returns a NodeName.
func NewNodeName(s string) (NodeName, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed != s {
		return NodeName{}, domainerr.ErrInvalidNodeName
	}
	if trimmed == "" {
		return NodeName{}, domainerr.ErrInvalidNodeName
	}
	if utf8.RuneCountInString(trimmed) > maxNodeNameRunes {
		return NodeName{}, domainerr.ErrInvalidNodeName
	}
	if strings.ContainsRune(trimmed, '/') || strings.ContainsRune(trimmed, 0) {
		return NodeName{}, domainerr.ErrInvalidNodeName
	}
	if trimmed == "." || trimmed == ".." {
		return NodeName{}, domainerr.ErrInvalidNodeName
	}
	return NodeName{value: trimmed}, nil
}

// NodeNameFromTrusted wraps a string previously stored in DB without revalidation.
func NodeNameFromTrusted(s string) NodeName {
	return NodeName{value: s}
}

func (n NodeName) String() string             { return n.value }
func (n NodeName) IsZero() bool               { return n.value == "" }
func (n NodeName) Equals(other NodeName) bool { return n.value == other.value }
