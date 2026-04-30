package middleware

import (
	"net/http"

	"github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/infrastructure/config"
)

// CORS creates a middleware that handles CORS headers.
func CORS(cfg config.CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", cfg.AllowOrigins)
			w.Header().Set("Access-Control-Allow-Methods", cfg.AllowMethods)
			w.Header().Set("Access-Control-Allow-Headers", cfg.AllowHeaders)

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
