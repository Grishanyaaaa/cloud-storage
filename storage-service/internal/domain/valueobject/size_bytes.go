package valueobject

import (
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

// MaxFileSizeBytes is the absolute hard upper bound on file size in bytes.
// 5 GiB per TZ. The application also enforces a configurable limit
// (StorageConfig.MaxFileSizeBytes) which must be ≤ this constant.
const MaxFileSizeBytes int64 = 5 * 1024 * 1024 * 1024

// SizeBytes is a validated non-negative byte count.
type SizeBytes struct {
	value int64
}

// NewSizeBytes validates the supplied raw size and returns a SizeBytes.
//   - must be ≥ 0
//   - must be ≤ MaxFileSizeBytes
func NewSizeBytes(v int64) (SizeBytes, error) {
	if v < 0 {
		return SizeBytes{}, domainerr.ErrInvalidSize
	}
	if v > MaxFileSizeBytes {
		return SizeBytes{}, domainerr.ErrFileTooLarge
	}
	return SizeBytes{value: v}, nil
}

// SizeBytesFromTrusted wraps a value previously stored in DB without revalidation.
func SizeBytesFromTrusted(v int64) SizeBytes {
	return SizeBytes{value: v}
}

func (s SizeBytes) Value() int64               { return s.value }
func (s SizeBytes) IsZero() bool               { return s.value == 0 }
func (s SizeBytes) Equals(other SizeBytes) bool { return s.value == other.value }

// MiB returns the size rounded up to whole mebibytes (used by TTL policy).
func (s SizeBytes) MiB() int64 {
	const mib = 1024 * 1024
	if s.value <= 0 {
		return 0
	}
	return (s.value + mib - 1) / mib
}
