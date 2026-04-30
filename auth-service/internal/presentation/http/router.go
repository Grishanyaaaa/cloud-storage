package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/config"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/presentation/http/handler"
	custommiddleware "github.com/Grishanyaaaa/cloud-storage/auth-service/internal/presentation/http/middleware"
)

func NewRouter(authHandler *handler.AuthHandler, corsConfig config.CORSConfig) (*chi.Mux, *custommiddleware.RateLimiter) {
	r := chi.NewRouter()

	// Базовые мидлвари
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(custommiddleware.CORS(corsConfig))

	// Rate limiting: 10 requests per second with burst of 20
	rateLimiter := custommiddleware.NewRateLimiter(rate.Limit(10), 20)
	r.Use(rateLimiter.Middleware)

	// Эндпоинты аутентификации
	r.Route("/auth/v1", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	// JWKS эндпоинт для Gateway и других сервисов
	r.Get("/.well-known/jwks.json", authHandler.GetJWKS)

	return r, rateLimiter
}
