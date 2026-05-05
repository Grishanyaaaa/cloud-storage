package port

import "net/http"

// JWTPayload is the subset of JWT claims used by ai-service.
// Issued by auth-service; ai-service only verifies and reads — never signs.
type JWTPayload struct {
	UserID   string
	Email    string
	Roles    []string
	RawToken string // verbatim Bearer token; propagated to storage-service
}

// JWTParser validates and extracts JWTPayload from an incoming request.
// On any failure (missing header, bad signature, expired, wrong issuer/audience)
// it returns a domainerr (ErrUnauthorized / ErrInvalidToken / ErrTokenExpired).
type JWTParser interface {
	Parse(r *http.Request) (*JWTPayload, error)
}
