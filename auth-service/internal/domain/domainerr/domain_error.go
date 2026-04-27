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
	// Используем errors.AsType (Go 1.26+) для проверки
	if de, ok := errors.AsType[*DomainError](err); ok {
		return de.Code == "USER_NOT_FOUND" || de.Code == "REFRESH_TOKEN_NOT_FOUND" || de.Code == "AUDIT_LOG_NOT_FOUND"
	}
	return false
}
