package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
	Code   string      `json:"code,omitempty"`
}

func SendSuccess(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(Response{Status: "success", Data: data}); err != nil {
		slog.Error("failed to encode success response", "error", err)
	}
}

func SendNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// SendError maps domain / dto errors to HTTP status codes per blueprint §11.
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
	case domainerr.IsGone(err):
		code = http.StatusGone
		message = err.Error()
	case errors.Is(err, domainerr.ErrUnauthorized),
		errors.Is(err, domainerr.ErrInvalidToken),
		errors.Is(err, domainerr.ErrTokenExpired):
		code = http.StatusUnauthorized
		message = err.Error()
	case errors.Is(err, domainerr.ErrFileTooLarge):
		code = http.StatusRequestEntityTooLarge
		message = err.Error()
	case errors.Is(err, domainerr.ErrRootImmutable),
		errors.Is(err, domainerr.ErrFileNotPending),
		errors.Is(err, domainerr.ErrFileNotActive),
		errors.Is(err, domainerr.ErrNodeAlreadyDeleted),
		errors.Is(err, domainerr.ErrMoveIntoSelf),
		errors.Is(err, domainerr.ErrMoveAcrossOwners),
		errors.Is(err, domainerr.ErrNodeKindMismatch):
		code = http.StatusUnprocessableEntity
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

// isClientValidationError covers DTO-level and value-object validation errors
// surfaced before reaching the use case layer.
func isClientValidationError(err error) bool {
	switch {
	case errors.Is(err, dto.ErrNameRequired),
		errors.Is(err, dto.ErrParentIDRequired),
		errors.Is(err, dto.ErrNodeIDRequired),
		errors.Is(err, dto.ErrSizeRequired),
		errors.Is(err, dto.ErrInvalidLimit),
		errors.Is(err, dto.ErrInvalidPermission),
		errors.Is(err, dto.ErrInvalidExpiresIn),
		errors.Is(err, dto.ErrEmptyChecksum),
		errors.Is(err, dto.ErrInvalidChecksum),
		errors.Is(err, dto.ErrInvalidDisposition),
		errors.Is(err, domainerr.ErrInvalidNodeID),
		errors.Is(err, domainerr.ErrInvalidShareID),
		errors.Is(err, domainerr.ErrInvalidShareToken),
		errors.Is(err, domainerr.ErrInvalidNodeName),
		errors.Is(err, domainerr.ErrInvalidPermission),
		errors.Is(err, domainerr.ErrInvalidMimeType),
		errors.Is(err, domainerr.ErrInvalidSize):
		return true
	}
	return false
}
