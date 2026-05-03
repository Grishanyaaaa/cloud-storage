package valueobject

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

const (
	// ShareTokenRawBytes is the raw entropy length of a share token (32 bytes ≈ 256 bits).
	ShareTokenRawBytes = 32
	// shareTokenEncodedLen is the length of base64.RawURLEncoding-encoded ShareTokenRawBytes (43).
	shareTokenEncodedLen = 43
	// shareTokenHashLen is the length of sha256(token) in hex.
	shareTokenHashLen = 64
)

// ShareToken is a base64url-encoded random token shown to the user once at creation time.
// We never persist the raw token — only sha256(token) hex.
type ShareToken struct {
	value string // base64url-encoded raw bytes (43 chars)
}

// ParseShareToken validates a token presented in a public route URL.
func ParseShareToken(s string) (ShareToken, error) {
	if len(s) != shareTokenEncodedLen {
		return ShareToken{}, domainerr.ErrInvalidShareToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(raw) != ShareTokenRawBytes {
		return ShareToken{}, domainerr.ErrInvalidShareToken
	}
	return ShareToken{value: s}, nil
}

// ShareTokenFromTrusted wraps a freshly generated raw byte slice without revalidation.
// Used by the security adapter that produces tokens.
func ShareTokenFromBytes(raw []byte) (ShareToken, error) {
	if len(raw) != ShareTokenRawBytes {
		return ShareToken{}, domainerr.ErrInvalidShareToken
	}
	return ShareToken{value: base64.RawURLEncoding.EncodeToString(raw)}, nil
}

func (t ShareToken) String() string { return t.value }

// Hash computes sha256(token) hex — the value persisted in DB.
func (t ShareToken) Hash() string {
	sum := sha256.Sum256([]byte(t.value))
	return hex.EncodeToString(sum[:])
}

// IsZero reports whether the token is the zero value.
func (t ShareToken) IsZero() bool { return t.value == "" }

// HashTokenString hashes a raw user-supplied token string (after format check)
// without constructing a ShareToken — handy for repository lookups.
func HashTokenString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// IsValidTokenHash returns true when s looks like a 64-char lowercase hex hash.
func IsValidTokenHash(s string) bool {
	if len(s) != shareTokenHashLen {
		return false
	}
	return strings.IndexFunc(s, func(r rune) bool {
		return !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f'))
	}) == -1
}
