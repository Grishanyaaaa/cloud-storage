package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/presentation/http/handler"
)

func NewRouter(authHandler *handler.AuthHandler) *chi.Mux {
	r := chi.NewRouter()

	// Базовые мидлвари
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)

	// Эндпоинты аутентификации
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	// JWKS эндпоинт для Gateway и других сервисов
	r.Get("/.well-known/jwks.json", authHandler.GetJWKS)

	return r
}
