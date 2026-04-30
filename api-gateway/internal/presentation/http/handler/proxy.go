package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProxyHandler handles reverse proxy requests to upstream services.
type ProxyHandler struct {
	authServiceURL string
	httpClient     *http.Client
}

// NewProxyHandler creates a new proxy handler.
func NewProxyHandler(authServiceURL string) *ProxyHandler {
	return &ProxyHandler{
		authServiceURL: authServiceURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProxyToAuthService forwards requests to auth-service.
func (h *ProxyHandler) ProxyToAuthService(w http.ResponseWriter, r *http.Request) {
	// Build upstream URL
	upstreamURL, err := url.Parse(h.authServiceURL)
	if err != nil {
		http.Error(w, `{"status":"error","error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Preserve the original path
	upstreamURL.Path = r.URL.Path
	upstreamURL.RawQuery = r.URL.RawQuery

	// Create new request
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), r.Body)
	if err != nil {
		http.Error(w, `{"status":"error","error":"failed to create proxy request"}`, http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		// Skip hop-by-hop headers
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set X-Forwarded headers
	if r.RemoteAddr != "" {
		proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
	}
	proxyReq.Header.Set("X-Forwarded-Proto", "http")
	proxyReq.Header.Set("X-Forwarded-Host", r.Host)

	// Execute request
	resp, err := h.httpClient.Do(proxyReq)
	if err != nil {
		http.Error(w, `{"status":"error","error":"upstream service unavailable"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		// Log error but can't change response status (headers already sent)
		fmt.Printf("error copying response body: %v\n", err)
	}
}

// isHopByHopHeader checks if a header is hop-by-hop.
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	header = strings.ToLower(header)
	for _, h := range hopByHopHeaders {
		if strings.ToLower(h) == header {
			return true
		}
	}
	return false
}
