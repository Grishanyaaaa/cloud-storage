package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/domainerr"
)

type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func SendSuccess(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   data,
	})
}

func SendError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	code := http.StatusInternalServerError
	message := "internal server error"

	// Маппинг доменных ошибок в HTTP статусы
	if domainerr.IsNotFound(err) {
		code = http.StatusNotFound
		message = err.Error()
	} else if errors.Is(err, domainerr.ErrInvalidCredentials) ||
		errors.Is(err, domainerr.ErrInvalidToken) ||
		errors.Is(err, domainerr.ErrTokenExpired) ||
		errors.Is(err, domainerr.ErrRefreshTokenRevoked) {
		code = http.StatusUnauthorized
		message = err.Error()
	} else if errors.Is(err, domainerr.ErrUserAlreadyExists) {
		code = http.StatusConflict
		message = err.Error()
	} else if errors.Is(err, domainerr.ErrUserInactive) {
		code = http.StatusForbidden
		message = err.Error()
	} else if errors.Is(err, dto.ErrEmailRequired) ||
		errors.Is(err, dto.ErrPasswordRequired) ||
		errors.Is(err, dto.ErrRefreshTokenRequired) {
		code = http.StatusBadRequest
		message = err.Error()
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(Response{
		Status: "error",
		Error:  message,
	})
}
