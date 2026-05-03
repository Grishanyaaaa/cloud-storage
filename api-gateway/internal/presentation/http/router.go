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

	// Public auth routes - proxy to auth-service without JWT validation
	r.Route("/auth", func(r chi.Router) {
		r.HandleFunc("/v1/register", proxyHandler.ProxyToAuthService)
		r.HandleFunc("/v1/login", proxyHandler.ProxyToAuthService)
		r.HandleFunc("/v1/refresh", proxyHandler.ProxyToAuthService)
		r.HandleFunc("/v1/logout", proxyHandler.ProxyToAuthService)
	})

	// JWKS endpoint - proxy to auth-service
	r.Get("/.well-known/jwks.json", proxyHandler.ProxyToAuthService)

	// Public storage share routes - proxy to storage-service without JWT validation
	// (authentication is performed via the share token in the URL path).
	r.Route("/storage/v1/public/{token}", func(r chi.Router) {
		r.Get("/", proxyHandler.ProxyToStorageService)
		r.Get("/tree", proxyHandler.ProxyToStorageService)
		r.Get("/folders/{id}/children", proxyHandler.ProxyToStorageService)
		r.Get("/nodes/{id}", proxyHandler.ProxyToStorageService)
		r.Get("/files/{id}/download-url", proxyHandler.ProxyToStorageService)
		r.Patch("/nodes/{id}/rename", proxyHandler.ProxyToStorageService)
		r.Delete("/nodes/{id}", proxyHandler.ProxyToStorageService)
	})

	// Protected routes - require JWT validation
	r.Route("/api", func(r chi.Router) {
		r.Use(custommiddleware.JWTAuth(jwksClient, cfg.JWT.Issuer, cfg.JWT.Audience))

		// Future protected endpoints will be added here
		// Example: r.HandleFunc("/v1/profile", proxyHandler.ProxyToUserService)
	})

	// Protected storage routes - require JWT validation, proxy to storage-service.
	r.Route("/storage/v1", func(r chi.Router) {
		r.Use(custommiddleware.JWTAuth(jwksClient, cfg.JWT.Issuer, cfg.JWT.Audience))

		r.Post("/me/root", proxyHandler.ProxyToStorageService)
		r.Get("/tree", proxyHandler.ProxyToStorageService)

		r.Route("/folders", func(r chi.Router) {
			r.Post("/", proxyHandler.ProxyToStorageService)
			r.Get("/{id}/children", proxyHandler.ProxyToStorageService)
		})

		r.Route("/nodes/{id}", func(r chi.Router) {
			r.Get("/", proxyHandler.ProxyToStorageService)
			r.Patch("/rename", proxyHandler.ProxyToStorageService)
			r.Patch("/move", proxyHandler.ProxyToStorageService)
			r.Delete("/", proxyHandler.ProxyToStorageService)
			r.Post("/restore", proxyHandler.ProxyToStorageService)
			r.Post("/shares", proxyHandler.ProxyToStorageService)
			r.Get("/shares", proxyHandler.ProxyToStorageService)
		})

		r.Route("/files", func(r chi.Router) {
			r.Post("/upload-url", proxyHandler.ProxyToStorageService)
			r.Post("/{id}/finalize", proxyHandler.ProxyToStorageService)
			r.Post("/{id}/abort", proxyHandler.ProxyToStorageService)
			r.Get("/{id}/download-url", proxyHandler.ProxyToStorageService)
		})

		r.Delete("/shares/{id}", proxyHandler.ProxyToStorageService)
	})

	return r, rateLimiter
}
