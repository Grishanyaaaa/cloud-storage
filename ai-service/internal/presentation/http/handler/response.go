package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
)

// Response is the canonical envelope returned by every handler.
type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
	Code   string      `json:"code,omitempty"`
}

// SendSuccess writes a 2xx JSON envelope.
func SendSuccess(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Response{Status: "success", Data: data}); err != nil {
		slog.Error("failed to encode success response", "error", err)
	}
}

// SendNoContent writes 204.
func SendNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// SendError maps domain / dto errors to HTTP status codes.
//
// Mapping (per ai-service blueprint §11):
//
//	NotFound        → 404
//	Conflict        → 409  (already executed/cancelled, expired)
//	Forbidden       → 403
//	Unauthorized    → 401
//	BadRequest      → 400  (validation, bad ids, bad ops)
//	UpstreamUnavail → 502  (LLM / storage-service down)
//	default         → 500
func SendError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	code := http.StatusInternalServerError
	message := "internal server error"
	domainCode := ""

	switch {
	case domainerr.IsNotFound(err):
		code = http.StatusNotFound
		message = err.Error()
	case domainerr.IsConflict(err):
		code = http.StatusConflict
		message = err.Error()
	case domainerr.IsForbidden(err):
		code = http.StatusForbidden
		message = err.Error()
	case domainerr.IsUnauthorized(err):
		code = http.StatusUnauthorized
		message = err.Error()
	case domainerr.IsUpstreamUnavailable(err):
		code = http.StatusBadGateway
		message = err.Error()
	case domainerr.IsBadRequest(err):
		code = http.StatusBadRequest
		message = err.Error()
	case isClientValidationError(err):
		code = http.StatusBadRequest
		message = err.Error()
	default:
		var de *domainerr.DomainError
		if errors.As(err, &de) {
			code = http.StatusBadRequest
			message = de.Error()
			domainCode = de.Code
		} else {
			slog.Error("unhandled error in handler", "error", err)
		}
	}

	if domainCode == "" {
		var de *domainerr.DomainError
		if errors.As(err, &de) {
			domainCode = de.Code
		}
	}

	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(Response{
		Status: "error",
		Error:  message,
		Code:   domainCode,
	}); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

// isClientValidationError matches DTO-level validation errors (sentinel pkg-level vars).
func isClientValidationError(err error) bool {
	switch {
	case errors.Is(err, dto.ErrInputRequired),
		errors.Is(err, dto.ErrCommandIDRequired),
		errors.Is(err, dto.ErrInvalidCommandID):
		return true
	}
	return false
}
