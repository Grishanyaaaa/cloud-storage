package domainerr

// User errors.
var (
	ErrUserNotFound       = New("USER_NOT_FOUND", "user not found", nil)
	ErrUserAlreadyExists  = New("USER_ALREADY_EXISTS", "user already exists", nil)
	ErrUserInactive       = New("USER_INACTIVE", "user account is deactivated", nil)
	ErrInvalidCredentials = New("INVALID_CREDENTIALS", "invalid email or password", nil)
)

// Token errors.
var (
	ErrInvalidToken         = New("INVALID_TOKEN", "invalid token", nil)
	ErrTokenExpired         = New("TOKEN_EXPIRED", "token expired", nil)
	ErrRefreshTokenNotFound = New("REFRESH_TOKEN_NOT_FOUND", "refresh token not found", nil)
	ErrRefreshTokenRevoked  = New("REFRESH_TOKEN_REVOKED", "refresh token revoked", nil)
)

var (
	ErrAuditLogNotFound = New("AUDIT_LOG_NOT_FOUND", "audit log not found", nil)
)
