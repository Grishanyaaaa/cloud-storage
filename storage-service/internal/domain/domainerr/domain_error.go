package domainerr

import "errors"

// DomainError represents a domain-specific domainerr with additional context.
// Code is used for mapping to HTTP/gRPC status codes in the transport layer.
type DomainError struct {
	Code    string
	Message string
	Cause   error
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

func New(code, message string, cause error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// IsNotFound checks if the error is a "NOT_FOUND" domain error.
func IsNotFound(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeNodeNotFound,
			CodeFileBlobNotFound,
			CodeUserRootNotFound,
			CodeShareNotFound:
			return true
		}
	}
	return false
}

// IsConflict checks if the error is a conflict (duplicate/violated constraint).
func IsConflict(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeNodeNameTaken,
			CodeUserRootAlreadyExists:
			return true
		}
	}
	return false
}

// IsForbidden checks if the error is an authorization failure.
func IsForbidden(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeForbidden,
			CodeShareScopeViolation,
			CodePermissionDenied:
			return true
		}
	}
	return false
}

// IsGone checks if the error indicates a resource is no longer available.
func IsGone(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeShareRevoked, CodeShareExpired:
			return true
		}
	}
	return false
}
