package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	authServiceURL    string
	storageServiceURL string
	httpClient        *http.Client
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(authServiceURL, storageServiceURL string) *HealthHandler {
	return &HealthHandler{
		authServiceURL:    authServiceURL,
		storageServiceURL: storageServiceURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type healthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services,omitempty"`
}

// Health returns basic health status (always returns 200 if gateway is running).
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(healthResponse{
		Status: "ok",
	})
}

// Ready checks if gateway and upstream services are ready.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	services := make(map[string]string)
	allHealthy := true

	authStatus := h.checkURL(ctx, h.authServiceURL+"/.well-known/jwks.json")
	services["auth-service"] = authStatus
	if authStatus != "healthy" {
		allHealthy = false
	}

	storageStatus := h.checkURL(ctx, h.storageServiceURL+"/healthz")
	services["storage-service"] = storageStatus
	if storageStatus != "healthy" {
		allHealthy = false
	}

	w.Header().Set("Content-Type", "application/json")

	if allHealthy {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(healthResponse{
			Status:   "ready",
			Services: services,
		})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(healthResponse{
			Status:   "not ready",
			Services: services,
		})
	}
}

// checkURL performs an HTTP GET to the given URL and returns "healthy" on 200.
func (h *HealthHandler) checkURL(ctx context.Context, url string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "unhealthy"
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "unhealthy"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "healthy"
	}
	return "unhealthy"
}
