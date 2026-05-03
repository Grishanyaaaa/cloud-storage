package dto

import "errors"

// Validation errors for DTO fields.
var (
	ErrNameRequired       = errors.New("name is required")
	ErrParentIDRequired   = errors.New("parent_id is required")
	ErrNodeIDRequired     = errors.New("node_id is required")
	ErrSizeRequired       = errors.New("size_bytes is required and must be > 0")
	ErrInvalidLimit       = errors.New("invalid limit")
	ErrInvalidPermission  = errors.New("invalid permission (must be 'view' or 'edit')")
	ErrInvalidExpiresIn   = errors.New("invalid expires_in")
	ErrEmptyChecksum      = errors.New("checksum is required")
	ErrInvalidChecksum    = errors.New("checksum must be 64 lowercase hex chars (sha256)")
	ErrInvalidDisposition = errors.New("disposition must be 'inline' or 'attachment'")
)
