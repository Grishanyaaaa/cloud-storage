package port

import "net/http"

// JWTPayload is the subset of JWT claims used by storage-service.
// Issued by auth-service; storage-service only verifies and reads.
type JWTPayload struct {
	UserID string
	Email  string
	Roles  []string
}

// JWTParser validates and extracts JWTPayload from an incoming request.
// On any failure (missing header, bad signature, expired, wrong issuer/audience)
// it returns a domainerr (ErrUnauthorized / ErrInvalidToken / ErrTokenExpired).
type JWTParser interface {
	Parse(r *http.Request) (*JWTPayload, error)
}
