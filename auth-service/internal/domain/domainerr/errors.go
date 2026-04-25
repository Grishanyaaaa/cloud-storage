package domainerr

import "errors"

// User errors.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserInactive       = errors.New("user account is deactivated")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// Token errors.
var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrTokenExpired         = errors.New("token expired")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenRevoked  = errors.New("refresh token revoked")
)

var (
	ErrAuditLogNotFound = errors.New("audit log not found")
)
