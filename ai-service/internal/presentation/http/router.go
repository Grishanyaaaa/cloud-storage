package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/config"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/presentation/http/handler"
	custommiddleware "github.com/Grishanyaaaa/cloud-storage/ai-service/internal/presentation/http/middleware"
)

// ActorExtractor for handler/middleware decoupling. Re-exported here for the
// composition root (cmd/ai-service/main.go) to provide.
type ActorExtractor = handler.ActorExtractor

// DefaultActorExtractor returns the actor previously stored by AuthContext middleware.
func DefaultActorExtractor(r *http.Request) *port.Actor {
	return custommiddleware.ActorFromContext(r.Context())
}

// NewRouter wires up the chi router for ai-service.
//
// Layout:
//
//	/ai/v1/commands              — POST  Plan command
//	/ai/v1/commands/{id}         — GET   Get command
//	/ai/v1/commands/{id}/execute — POST  Execute confirmed plan
//	/ai/v1/commands/{id}/cancel  — POST  Cancel pending plan
//	/healthz                     — liveness
//	/readyz                      — readiness
func NewRouter(
	aiHandler *handler.AIHandler,
	jwtParser port.JWTParser,
	corsCfg config.CORSConfig,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.CleanPath)
	// CORS is handled by api-gateway, no need for it here

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	r.Route("/ai/v1", func(r chi.Router) {
		r.Use(custommiddleware.AuthContext(jwtParser))

		r.Route("/commands", func(r chi.Router) {
			r.Post("/", aiHandler.PlanCommand)
			r.Get("/{id}", aiHandler.GetCommand)
			r.Post("/{id}/execute", aiHandler.ExecuteCommand)
			r.Post("/{id}/cancel", aiHandler.CancelCommand)
		})
	})

	return r
}
