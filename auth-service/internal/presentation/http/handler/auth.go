package handler

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/port"
)

type AuthHandler struct {
	useCase      port.AuthUseCase
	tokenManager port.TokenManager
}

func NewAuthHandler(useCase port.AuthUseCase, tokenManager port.TokenManager) *AuthHandler {
	return &AuthHandler{
		useCase:      useCase,
		tokenManager: tokenManager,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		req.IPAddress = host
	} else {
		req.IPAddress = r.RemoteAddr
	}
	req.UserAgent = r.UserAgent()

	resp, err := h.useCase.Register(r.Context(), req)
	if err != nil {
		SendError(w, err)
		return
	}

	SendSuccess(w, resp, http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		req.IPAddress = host
	} else {
		req.IPAddress = r.RemoteAddr
	}
	req.UserAgent = r.UserAgent()

	resp, err := h.useCase.Login(r.Context(), req)
	if err != nil {
		SendError(w, err)
		return
	}

	SendSuccess(w, resp, http.StatusOK)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		req.IPAddress = host
	} else {
		req.IPAddress = r.RemoteAddr
	}
	req.UserAgent = r.UserAgent()

	resp, err := h.useCase.Refresh(r.Context(), req)
	if err != nil {
		SendError(w, err)
		return
	}

	SendSuccess(w, resp, http.StatusOK)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		req.IPAddress = host
	} else {
		req.IPAddress = r.RemoteAddr
	}
	req.UserAgent = r.UserAgent()

	if err := h.useCase.Logout(r.Context(), req); err != nil {
		SendError(w, err)
		return
	}

	SendSuccess(w, nil, http.StatusNoContent)
}

func (h *AuthHandler) GetJWKS(w http.ResponseWriter, r *http.Request) {
	jwks, err := h.tokenManager.GetJWKS()
	if err != nil {
		SendError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jwks)
}
