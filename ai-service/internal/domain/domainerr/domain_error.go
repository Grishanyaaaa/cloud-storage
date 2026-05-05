package domainerr

import "errors"

// DomainError represents a domain-specific error with additional context.
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

// IsNotFound checks if the error is a NOT_FOUND domain error.
func IsNotFound(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeCommandNotFound:
			return true
		}
	}
	return false
}

// IsConflict checks if the error denotes a state-conflict (already executed / cancelled / expired).
func IsConflict(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeCommandAlreadyExecuted,
			CodeCommandAlreadyCancelled,
			CodeCommandExpired:
			return true
		}
	}
	return false
}

// IsForbidden checks if the error indicates an authorization failure
// (not-the-owner, or general forbidden state).
func IsForbidden(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeForbidden, CodeCommandForbidden:
			return true
		}
	}
	return false
}

// IsBadRequest checks if the error indicates malformed input or invalid plan.
func IsBadRequest(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeBadRequest,
			CodeInvalidCommandID,
			CodeInvalidUserID,
			CodeInvalidNodeID,
			CodeInvalidOperationKind,
			CodeInvalidCommandStatus,
			CodeInvalidPlanInput,
			CodeInputTooLong,
			CodeInputEmpty,
			CodeInvalidOperation,
			CodeInvalidPlan:
			return true
		}
	}
	return false
}

// IsUpstreamUnavailable checks if the error indicates an upstream dependency
// failure (storage-service or LLM provider).
func IsUpstreamUnavailable(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeLLMUnavailable,
			CodeLLMInvalidResponse,
			CodeStorageServiceUnavailable:
			return true
		}
	}
	return false
}

// IsUnauthorized checks if the error denotes a missing/invalid auth context.
func IsUnauthorized(err error) bool {
	if de, ok := errors.AsType[*DomainError](err); ok {
		switch de.Code {
		case CodeUnauthorized, CodeInvalidToken, CodeTokenExpired:
			return true
		}
	}
	return false
}
