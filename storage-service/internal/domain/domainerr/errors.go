package domainerr

// Error codes (kept as constants for handler mapping & tests).
const (
	// Node-related
	CodeNodeNotFound        = "NODE_NOT_FOUND"
	CodeNodeNameTaken       = "NODE_NAME_TAKEN"
	CodeNodeKindMismatch    = "NODE_KIND_MISMATCH"
	CodeNodeAlreadyDeleted  = "NODE_ALREADY_DELETED"
	CodeMoveIntoSelf        = "MOVE_INTO_SELF"
	CodeMoveAcrossOwners    = "MOVE_ACROSS_OWNERS"
	CodeRootImmutable       = "ROOT_IMMUTABLE"
	CodeInvalidNodeID       = "INVALID_NODE_ID"
	CodeInvalidNodeName     = "INVALID_NODE_NAME"
	CodeInvalidNodeKind     = "INVALID_NODE_KIND"
	CodeInvalidNodePath     = "INVALID_NODE_PATH"

	// File / blob-related
	CodeFileBlobNotFound = "FILE_BLOB_NOT_FOUND"
	CodeFileNotPending   = "FILE_NOT_PENDING"
	CodeFileNotActive    = "FILE_NOT_ACTIVE"
	CodeFileTooLarge     = "FILE_TOO_LARGE"
	CodeInvalidMimeType  = "INVALID_MIME_TYPE"
	CodeInvalidSize      = "INVALID_SIZE"
	CodeInvalidStorageKey = "INVALID_STORAGE_KEY"

	// User-root
	CodeUserRootNotFound      = "USER_ROOT_NOT_FOUND"
	CodeUserRootAlreadyExists = "USER_ROOT_ALREADY_EXISTS"
	CodeInvalidUserID         = "INVALID_USER_ID"

	// Share-related
	CodeShareNotFound       = "SHARE_NOT_FOUND"
	CodeShareRevoked        = "SHARE_REVOKED"
	CodeShareExpired        = "SHARE_EXPIRED"
	CodeShareScopeViolation = "SHARE_SCOPE_VIOLATION"
	CodeInvalidPermission   = "INVALID_PERMISSION"
	CodeInvalidShareID      = "INVALID_SHARE_ID"
	CodeInvalidShareToken   = "INVALID_SHARE_TOKEN"
	CodeInvalidExpiry       = "INVALID_EXPIRY"

	// Auth / authorization
	CodeForbidden        = "FORBIDDEN"
	CodePermissionDenied = "PERMISSION_DENIED"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeInvalidToken     = "INVALID_TOKEN"
	CodeTokenExpired     = "TOKEN_EXPIRED"

	// Generic / validation
	CodeBadRequest = "BAD_REQUEST"
)

// Node errors.
var (
	ErrNodeNotFound       = New(CodeNodeNotFound, "node not found", nil)
	ErrNodeNameTaken      = New(CodeNodeNameTaken, "node name already taken in this folder", nil)
	ErrNodeKindMismatch   = New(CodeNodeKindMismatch, "node kind does not match expected kind", nil)
	ErrNodeAlreadyDeleted = New(CodeNodeAlreadyDeleted, "node is already deleted", nil)
	ErrMoveIntoSelf       = New(CodeMoveIntoSelf, "cannot move node into itself or its descendant", nil)
	ErrMoveAcrossOwners   = New(CodeMoveAcrossOwners, "cannot move node across owners", nil)
	ErrRootImmutable      = New(CodeRootImmutable, "root folder cannot be renamed, moved or deleted", nil)
	ErrInvalidNodeID      = New(CodeInvalidNodeID, "invalid node id", nil)
	ErrInvalidNodeName    = New(CodeInvalidNodeName, "invalid node name", nil)
	ErrInvalidNodeKind    = New(CodeInvalidNodeKind, "invalid node kind", nil)
	ErrInvalidNodePath    = New(CodeInvalidNodePath, "invalid node path", nil)
)

// File / blob errors.
var (
	ErrFileBlobNotFound  = New(CodeFileBlobNotFound, "file blob not found", nil)
	ErrFileNotPending    = New(CodeFileNotPending, "file blob is not in pending state", nil)
	ErrFileNotActive     = New(CodeFileNotActive, "file blob is not active", nil)
	ErrFileTooLarge      = New(CodeFileTooLarge, "file exceeds maximum allowed size", nil)
	ErrInvalidMimeType   = New(CodeInvalidMimeType, "invalid mime type", nil)
	ErrInvalidSize       = New(CodeInvalidSize, "invalid file size", nil)
	ErrInvalidStorageKey = New(CodeInvalidStorageKey, "invalid storage key", nil)
)

// User-root errors.
var (
	ErrUserRootNotFound      = New(CodeUserRootNotFound, "user root not found", nil)
	ErrUserRootAlreadyExists = New(CodeUserRootAlreadyExists, "user root already exists", nil)
	ErrInvalidUserID         = New(CodeInvalidUserID, "invalid user id", nil)
)

// Share errors.
var (
	ErrShareNotFound       = New(CodeShareNotFound, "share not found", nil)
	ErrShareRevoked        = New(CodeShareRevoked, "share is revoked", nil)
	ErrShareExpired        = New(CodeShareExpired, "share is expired", nil)
	ErrShareScopeViolation = New(CodeShareScopeViolation, "target node is outside of share scope", nil)
	ErrInvalidPermission   = New(CodeInvalidPermission, "invalid permission", nil)
	ErrInvalidShareID      = New(CodeInvalidShareID, "invalid share id", nil)
	ErrInvalidShareToken   = New(CodeInvalidShareToken, "invalid share token", nil)
	ErrInvalidExpiry       = New(CodeInvalidExpiry, "invalid expiry", nil)
)

// Auth errors.
var (
	ErrForbidden        = New(CodeForbidden, "forbidden", nil)
	ErrPermissionDenied = New(CodePermissionDenied, "permission denied", nil)
	ErrUnauthorized     = New(CodeUnauthorized, "unauthorized", nil)
	ErrInvalidToken     = New(CodeInvalidToken, "invalid token", nil)
	ErrTokenExpired     = New(CodeTokenExpired, "token expired", nil)
)

// Generic errors.
var (
	ErrBadRequest = New(CodeBadRequest, "bad request", nil)
)
