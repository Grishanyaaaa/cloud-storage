package valueobject

import (
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

// Permission enumerates supported share-link permissions.
type Permission string

const (
	PermissionView Permission = "view"
	PermissionEdit Permission = "edit"
)

// ParsePermission parses a string into a Permission.
func ParsePermission(s string) (Permission, error) {
	switch Permission(s) {
	case PermissionView, PermissionEdit:
		return Permission(s), nil
	default:
		return "", domainerr.ErrInvalidPermission
	}
}

func (p Permission) String() string { return string(p) }

// AllowsRead always true (view ⊂ edit).
func (p Permission) AllowsRead() bool { return p == PermissionView || p == PermissionEdit }

// AllowsRename only edit.
func (p Permission) AllowsRename() bool { return p == PermissionEdit }

// AllowsDelete only edit.
func (p Permission) AllowsDelete() bool { return p == PermissionEdit }
