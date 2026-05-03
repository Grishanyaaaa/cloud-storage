package valueobject

import (
	"mime"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

const (
	defaultMime    = "application/octet-stream"
	maxMimeTypeLen = 255
)

// MimeType is a validated MIME content type.
type MimeType struct {
	value string
}

// NewMimeType validates the supplied raw mime type and returns a MimeType.
// Empty or invalid input falls back to "application/octet-stream".
func NewMimeType(s string) (MimeType, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return MimeType{value: defaultMime}, nil
	}
	if len(s) > maxMimeTypeLen {
		return MimeType{}, domainerr.ErrInvalidMimeType
	}
	mt, _, err := mime.ParseMediaType(s)
	if err != nil || mt == "" {
		return MimeType{}, domainerr.ErrInvalidMimeType
	}
	return MimeType{value: mt}, nil
}

// MimeTypeFromTrusted wraps a string previously stored in DB without revalidation.
func MimeTypeFromTrusted(s string) MimeType {
	if s == "" {
		s = defaultMime
	}
	return MimeType{value: s}
}

// DefaultMimeType returns the fallback "application/octet-stream".
func DefaultMimeType() MimeType {
	return MimeType{value: defaultMime}
}

func (m MimeType) String() string             { return m.value }
func (m MimeType) IsZero() bool               { return m.value == "" }
func (m MimeType) Equals(other MimeType) bool { return m.value == other.value }
