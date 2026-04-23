package error

import "errors"

// User errors.
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserInactive       = errors.New("user account is deactivated")
	ErrInvalidCredentials = errors.New("invalid email or password")
)
