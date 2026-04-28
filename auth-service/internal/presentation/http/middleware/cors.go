package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/config"
)

// CORS middleware для обработки CORS запросов
func CORS(cfg config.CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Проверяем разрешенные origins
			if origin != "" && isOriginAllowed(origin, cfg.AllowOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Устанавливаем остальные CORS заголовки
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Обработка preflight запросов
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed проверяет, разрешен ли origin
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
