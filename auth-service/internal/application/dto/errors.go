package dto

import "errors"

// Validation errors for DTO fields.
var (
	ErrEmailRequired             = errors.New("email is required")
	ErrPasswordRequired          = errors.New("password is required")
	ErrRefreshTokenRequired      = errors.New("refresh_token is required")
	ErrInvalidRefreshTokenFormat = errors.New("invalid refresh_token format")
)
