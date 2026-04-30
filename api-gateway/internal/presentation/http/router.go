package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"

	"github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/infrastructure/client"
	"github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/infrastructure/config"
	"github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/presentation/http/handler"
	custommiddleware "github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/presentation/http/middleware"
)

func NewRouter(
	proxyHandler *handler.ProxyHandler,
	healthHandler *handler.HealthHandler,
	jwksClient *client.JWKSClient,
	cfg *config.Config,
) (*chi.Mux, *custommiddleware.RateLimiter) {
	r := chi.NewRouter()

	// Базовые мидлвари
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(custommiddleware.CORS(cfg.CORS))

	// Rate limiting: 100 requests per second with burst of 200
	rateLimiter := custommiddleware.NewRateLimiter(rate.Limit(100), 200)
	r.Use(rateLimiter.Middleware)

	// Health check endpoints (no auth required)
	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)

	// Public routes - proxy to auth-service without JWT validation
	r.Route("/auth", func(r chi.Router) {
		r.HandleFunc("/v1/register", proxyHandler.ProxyToAuthService)
		r.HandleFunc("/v1/login", proxyHandler.ProxyToAuthService)
		r.HandleFunc("/v1/refresh", proxyHandler.ProxyToAuthService)
		r.HandleFunc("/v1/logout", proxyHandler.ProxyToAuthService)
	})

	// JWKS endpoint - proxy to auth-service
	r.Get("/.well-known/jwks.json", proxyHandler.ProxyToAuthService)

	// Protected routes - require JWT validation
	r.Route("/api", func(r chi.Router) {
		r.Use(custommiddleware.JWTAuth(jwksClient, cfg.JWT.Issuer, cfg.JWT.Audience))

		// Future protected endpoints will be added here
		// Example: r.HandleFunc("/v1/profile", proxyHandler.ProxyToUserService)
	})

	return r, rateLimiter
}
