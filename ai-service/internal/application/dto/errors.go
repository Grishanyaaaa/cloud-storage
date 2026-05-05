package dto

import "errors"

// Validation errors for DTO fields.
var (
	ErrInputRequired      = errors.New("input is required")
	ErrCommandIDRequired  = errors.New("command_id is required")
	ErrInvalidCommandID   = errors.New("invalid command_id")
)
