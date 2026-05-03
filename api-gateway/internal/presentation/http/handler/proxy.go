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
	authServiceURL    string
	storageServiceURL string
	httpClient        *http.Client
}

// NewProxyHandler creates a new proxy handler.
func NewProxyHandler(authServiceURL, storageServiceURL string) *ProxyHandler {
	return &ProxyHandler{
		authServiceURL:    authServiceURL,
		storageServiceURL: storageServiceURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProxyToAuthService forwards requests to auth-service.
func (h *ProxyHandler) ProxyToAuthService(w http.ResponseWriter, r *http.Request) {
	h.proxy(w, r, h.authServiceURL)
}

// ProxyToStorageService forwards requests to storage-service.
func (h *ProxyHandler) ProxyToStorageService(w http.ResponseWriter, r *http.Request) {
	h.proxy(w, r, h.storageServiceURL)
}

// proxy is the shared reverse-proxy implementation.
// It preserves path, query, headers (minus hop-by-hop) and adds X-Forwarded-* headers.
func (h *ProxyHandler) proxy(w http.ResponseWriter, r *http.Request, upstream string) {
	upstreamURL, err := url.Parse(upstream)
	if err != nil {
		http.Error(w, `{"status":"error","error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	upstreamURL.Path = r.URL.Path
	upstreamURL.RawQuery = r.URL.RawQuery

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), r.Body)
	if err != nil {
		http.Error(w, `{"status":"error","error":"failed to create proxy request"}`, http.StatusInternalServerError)
		return
	}

	for key, values := range r.Header {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	if r.RemoteAddr != "" {
		proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
	}
	proxyReq.Header.Set("X-Forwarded-Proto", "http")
	proxyReq.Header.Set("X-Forwarded-Host", r.Host)

	resp, err := h.httpClient.Do(proxyReq)
	if err != nil {
		http.Error(w, `{"status":"error","error":"upstream service unavailable"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
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
