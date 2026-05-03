package valueobject

import (
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

const (
	rootPath        = "/"
	maxNodePathLen  = 4096
	pathSeparator   = "/"
)

// NodePath is the materialized path of a node, stored as
// "/{ownerID}" for the root and "{parentPath}/{nodeID}" for everything else.
// All segments except the leading "/" are UUID strings.
type NodePath struct {
	value string
}

// NewRootPath returns the canonical path of an owner's root folder.
// We deliberately use the owner's UUID rather than the root NodeID so the
// path can be computed without an extra DB lookup at root creation time.
func NewRootPath(owner UserID) NodePath {
	return NodePath{value: rootPath + owner.String()}
}

// AppendChild builds the path of a child node under parent.
func AppendChild(parent NodePath, child NodeID) (NodePath, error) {
	if parent.IsZero() {
		return NodePath{}, domainerr.ErrInvalidNodePath
	}
	candidate := parent.value + pathSeparator + child.String()
	if len(candidate) > maxNodePathLen {
		return NodePath{}, domainerr.ErrInvalidNodePath
	}
	return NodePath{value: candidate}, nil
}

// ParseNodePath validates a stored path and wraps it.
func ParseNodePath(s string) (NodePath, error) {
	if s == "" || !strings.HasPrefix(s, rootPath) || len(s) > maxNodePathLen {
		return NodePath{}, domainerr.ErrInvalidNodePath
	}
	return NodePath{value: s}, nil
}

// NodePathFromTrusted wraps a string previously stored in DB without revalidation.
func NodePathFromTrusted(s string) NodePath {
	return NodePath{value: s}
}

func (p NodePath) String() string             { return p.value }
func (p NodePath) IsZero() bool               { return p.value == "" }
func (p NodePath) Equals(other NodePath) bool { return p.value == other.value }

// Depth returns the depth of the path counted from root (root depth = 1).
func (p NodePath) Depth() int {
	if p.value == "" {
		return 0
	}
	return strings.Count(p.value, pathSeparator)
}

// IsAncestorOf returns true when receiver covers `other` strictly (other is a descendant).
func (p NodePath) IsAncestorOf(other NodePath) bool {
	return strings.HasPrefix(other.value, p.value+pathSeparator)
}

// CoversOrEquals returns true when receiver equals or is an ancestor of other.
// Used by share scope checks: target must be inside (or equal to) the share root.
func (p NodePath) CoversOrEquals(other NodePath) bool {
	return p.value == other.value || p.IsAncestorOf(other)
}

// LikePrefix returns the SQL LIKE prefix for subtree queries:
//
//	WHERE path = $1 OR path LIKE $1 || '/%'
func (p NodePath) LikePrefix() string {
	return p.value + pathSeparator + "%"
}
