package valueobject

import (
	"fmt"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

const (
	maxStorageKeyLen = 1024
	storageKeyPrefix = "users/"
)

// StorageKey is the object key in the S3-compatible bucket.
//
// Layout: users/{ownerID}/{nodeID}.
// We do NOT include the user-visible name to avoid trouble on rename.
type StorageKey struct {
	value string
}

// NewStorageKey builds a key from owner and node IDs.
func NewStorageKey(owner UserID, node NodeID) (StorageKey, error) {
	if owner.IsZero() || node.IsZero() {
		return StorageKey{}, domainerr.ErrInvalidStorageKey
	}
	v := fmt.Sprintf("%s%s/%s", storageKeyPrefix, owner.String(), node.String())
	if len(v) > maxStorageKeyLen {
		return StorageKey{}, domainerr.ErrInvalidStorageKey
	}
	return StorageKey{value: v}, nil
}

// ParseStorageKey validates a stored key.
func ParseStorageKey(s string) (StorageKey, error) {
	if s == "" || len(s) > maxStorageKeyLen || !strings.HasPrefix(s, storageKeyPrefix) {
		return StorageKey{}, domainerr.ErrInvalidStorageKey
	}
	return StorageKey{value: s}, nil
}

// StorageKeyFromTrusted wraps a string previously stored in DB without revalidation.
func StorageKeyFromTrusted(s string) StorageKey {
	return StorageKey{value: s}
}

func (k StorageKey) String() string               { return k.value }
func (k StorageKey) IsZero() bool                 { return k.value == "" }
func (k StorageKey) Equals(other StorageKey) bool { return k.value == other.value }
