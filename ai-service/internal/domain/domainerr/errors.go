package domainerr

// Error codes (kept as constants for handler mapping & tests).
const (
	// Command lifecycle
	CodeCommandNotFound         = "COMMAND_NOT_FOUND"
	CodeCommandAlreadyExecuted  = "COMMAND_ALREADY_EXECUTED"
	CodeCommandAlreadyCancelled = "COMMAND_ALREADY_CANCELLED"
	CodeCommandExpired          = "COMMAND_EXPIRED"
	CodeCommandForbidden        = "COMMAND_FORBIDDEN"

	// Validation / VOs
	CodeInvalidCommandID     = "INVALID_COMMAND_ID"
	CodeInvalidUserID        = "INVALID_USER_ID"
	CodeInvalidNodeID        = "INVALID_NODE_ID"
	CodeInvalidOperationKind = "INVALID_OPERATION_KIND"
	CodeInvalidCommandStatus = "INVALID_COMMAND_STATUS"
	CodeInvalidPlanInput     = "INVALID_PLAN_INPUT"
	CodeInputTooLong         = "INPUT_TOO_LONG"
	CodeInputEmpty           = "INPUT_EMPTY"
	CodeInvalidOperation     = "INVALID_OPERATION"
	CodeInvalidPlan          = "INVALID_PLAN"

	// Auth / authorization
	CodeForbidden    = "FORBIDDEN"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeInvalidToken = "INVALID_TOKEN"
	CodeTokenExpired = "TOKEN_EXPIRED"

	// Upstream (LLM / storage-service)
	CodeLLMUnavailable            = "LLM_UNAVAILABLE"
	CodeLLMInvalidResponse        = "LLM_INVALID_RESPONSE"
	CodeStorageServiceUnavailable = "STORAGE_SERVICE_UNAVAILABLE"

	// Generic / validation
	CodeBadRequest = "BAD_REQUEST"
)

// Command lifecycle errors.
var (
	ErrCommandNotFound         = New(CodeCommandNotFound, "ai command not found", nil)
	ErrCommandAlreadyExecuted  = New(CodeCommandAlreadyExecuted, "ai command already executed", nil)
	ErrCommandAlreadyCancelled = New(CodeCommandAlreadyCancelled, "ai command already cancelled", nil)
	ErrCommandExpired          = New(CodeCommandExpired, "ai command plan expired", nil)
	ErrCommandForbidden        = New(CodeCommandForbidden, "ai command does not belong to caller", nil)
)

// Validation errors.
var (
	ErrInvalidCommandID     = New(CodeInvalidCommandID, "invalid ai command id", nil)
	ErrInvalidUserID        = New(CodeInvalidUserID, "invalid user id", nil)
	ErrInvalidNodeID        = New(CodeInvalidNodeID, "invalid node id", nil)
	ErrInvalidOperationKind = New(CodeInvalidOperationKind, "invalid operation kind", nil)
	ErrInvalidCommandStatus = New(CodeInvalidCommandStatus, "invalid command status", nil)
	ErrInvalidPlanInput     = New(CodeInvalidPlanInput, "invalid plan input", nil)
	ErrInputTooLong         = New(CodeInputTooLong, "input exceeds maximum allowed length", nil)
	ErrInputEmpty           = New(CodeInputEmpty, "input is empty", nil)
	ErrInvalidOperation     = New(CodeInvalidOperation, "operation is not valid against current tree", nil)
	ErrInvalidPlan          = New(CodeInvalidPlan, "plan is invalid or unsafe", nil)
)

// Auth errors.
var (
	ErrForbidden    = New(CodeForbidden, "operation forbidden", nil)
	ErrUnauthorized = New(CodeUnauthorized, "missing or invalid authentication", nil)
	ErrInvalidToken = New(CodeInvalidToken, "invalid token", nil)
	ErrTokenExpired = New(CodeTokenExpired, "token expired", nil)
)

// Upstream errors.
var (
	ErrLLMUnavailable            = New(CodeLLMUnavailable, "LLM provider unavailable", nil)
	ErrLLMInvalidResponse        = New(CodeLLMInvalidResponse, "LLM returned invalid response", nil)
	ErrStorageServiceUnavailable = New(CodeStorageServiceUnavailable, "storage-service unavailable", nil)
)

// Generic errors.
var (
	ErrBadRequest = New(CodeBadRequest, "bad request", nil)
)
