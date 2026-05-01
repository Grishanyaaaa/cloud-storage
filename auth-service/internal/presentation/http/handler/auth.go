package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/port"
)

type AuthHandler struct {
	useCase      port.AuthUseCase
	tokenManager port.TokenManager
	trustProxy   bool
}

func NewAuthHandler(useCase port.AuthUseCase, tokenManager port.TokenManager, trustProxy bool) *AuthHandler {
	return &AuthHandler{
		useCase:      useCase,
		tokenManager: tokenManager,
		trustProxy:   trustProxy,
	}
}

// extractClientInfo extracts IP address and User-Agent from the request.
// When trustProxy is true and behind a reverse proxy, it checks X-Forwarded-For and X-Real-IP headers.
// When trustProxy is false, only RemoteAddr is used to prevent IP spoofing.
func extractClientInfo(r *http.Request, trustProxy bool) (ip, userAgent string) {
	// Only trust proxy headers if explicitly configured
	if trustProxy {
		// Try X-Forwarded-For first (comma-separated list, first is client)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if idx := strings.Index(xff, ","); idx != -1 {
				ip = strings.TrimSpace(xff[:idx])
			} else {
				ip = strings.TrimSpace(xff)
			}
			// Validate IP from X-Forwarded-For
			if net.ParseIP(ip) != nil {
				// Valid IP found, use it
				goto extractUserAgent
			}
			ip = "" // Reset if invalid
		}

		// Fallback to X-Real-IP
		if ip == "" {
			if xri := r.Header.Get("X-Real-IP"); xri != "" {
				// Validate IP from X-Real-IP
				if net.ParseIP(xri) != nil {
					ip = xri
					goto extractUserAgent
				}
			}
		}
	}

	// Fallback to RemoteAddr (always used when trustProxy is false)
	if ip == "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ip = host
		} else {
			ip = r.RemoteAddr
		}
	}

	// Strip port if present (handles cases where proxy sends IP:PORT)
	if ip != "" {
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}
	}

	// Validate final IP
	if ip != "" && net.ParseIP(ip) == nil {
		// Invalid IP, leave empty (will be stored as NULL in DB)
		ip = ""
	}

extractUserAgent:
	// Leave empty if still no valid IP (will be stored as NULL in DB)
	// Don't use "unknown" as it's not a valid IP address

	userAgent = r.UserAgent()
	if len(userAgent) > 1024 {
		userAgent = truncateUTF8(userAgent, 1024) // Truncate safely without breaking UTF-8
	}
	if userAgent == "" {
		userAgent = "unknown"
	}

	return ip, userAgent
}

// truncateUTF8 truncates a string to maxBytes without breaking UTF-8 encoding.
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	// Find the last valid UTF-8 rune boundary before maxBytes
	for i := maxBytes; i > 0; i-- {
		if utf8.RuneStart(s[i]) {
			return s[:i]
		}
	}
	return ""
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	defer r.Body.Close()
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		SendError(w, err)
		return
	}

	req.IPAddress, req.UserAgent = extractClientInfo(r, h.trustProxy)

	resp, err := h.useCase.Register(r.Context(), req)
	if err != nil {
		SendError(w, err)
		return
	}

	SendSuccess(w, resp, http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	defer r.Body.Close()
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		SendError(w, err)
		return
	}

	req.IPAddress, req.UserAgent = extractClientInfo(r, h.trustProxy)

	resp, err := h.useCase.Login(r.Context(), req)
	if err != nil {
		SendError(w, err)
		return
	}

	SendSuccess(w, resp, http.StatusOK)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	defer r.Body.Close()
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		SendError(w, err)
		return
	}

	req.IPAddress, req.UserAgent = extractClientInfo(r, h.trustProxy)

	resp, err := h.useCase.Refresh(r.Context(), req)
	if err != nil {
		SendError(w, err)
		return
	}

	SendSuccess(w, resp, http.StatusOK)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	defer r.Body.Close()
	var req dto.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		SendError(w, err)
		return
	}

	req.IPAddress, req.UserAgent = extractClientInfo(r, h.trustProxy)

	if err := h.useCase.Logout(r.Context(), req); err != nil {
		SendError(w, err)
		return
	}

	// HTTP 204 No Content должен возвращаться без body
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) GetJWKS(w http.ResponseWriter, r *http.Request) {
	jwks, err := h.tokenManager.GetJWKS()
	if err != nil {
		SendError(w, err)
		return
	}

	// Marshal to buffer first to catch encoding errors before writing headers
	data, err := json.Marshal(jwks)
	if err != nil {
		SendError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		// Log the error but don't change response status (headers already sent)
		// This is a best-effort write; client will detect incomplete response
		return
	}
}
