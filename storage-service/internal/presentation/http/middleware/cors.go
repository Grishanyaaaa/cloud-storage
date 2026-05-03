package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/config"
)

// CORS handles preflight and origin headers.
func CORS(cfg config.CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && isOriginAllowed(origin, cfg.AllowOrigins, cfg.AllowCredentials) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isOriginAllowed(origin string, allowedOrigins []string, allowCredentials bool) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			if allowCredentials {
				return false
			}
			return true
		}
		if allowed == origin {
			return true
		}
	}
	return false
}
