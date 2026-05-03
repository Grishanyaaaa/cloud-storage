package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/config"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/presentation/http/handler"
	custommiddleware "github.com/Grishanyaaaa/cloud-storage/storage-service/internal/presentation/http/middleware"
)

// NewRouter wires up the chi router for storage-service.
//
// Layout:
//
//	/storage/v1                  — owner endpoints (JWT required)
//	/storage/v1/public/{token}/… — share-link endpoints (token resolved by middleware)
//	/healthz                     — liveness
//	/readyz                      — readiness
func NewRouter(
	storageHandler *handler.StorageHandler,
	useCase port.StorageUseCase,
	jwtParser port.JWTParser,
	corsCfg config.CORSConfig,
	serverCfg config.ServerConfig,
) (*chi.Mux, *custommiddleware.RateLimiter) {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.CleanPath)
	r.Use(custommiddleware.CORS(corsCfg))

	rateLimiter := custommiddleware.NewRateLimiter(rate.Limit(20), 40, serverCfg.TrustProxy)
	r.Use(rateLimiter.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	// ----- /storage/v1 (JWT required) -----
	r.Route("/storage/v1", func(r chi.Router) {
		r.Use(custommiddleware.AuthContext(jwtParser))

		r.Post("/me/root", storageHandler.EnsureRoot)
		r.Get("/tree", storageHandler.GetTree)

		r.Route("/folders", func(r chi.Router) {
			r.Post("/", storageHandler.CreateFolder)
			r.Get("/{id}/children", storageHandler.ListChildren)
		})

		r.Route("/nodes/{id}", func(r chi.Router) {
			r.Get("/", storageHandler.GetNode)
			r.Patch("/rename", storageHandler.RenameNode)
			r.Patch("/move", storageHandler.MoveNode)
			r.Delete("/", storageHandler.SoftDeleteNode)
			r.Post("/restore", storageHandler.RestoreNode)
			r.Post("/shares", storageHandler.CreateShareLink)
			r.Get("/shares", storageHandler.ListShareLinks)
		})

		r.Route("/files", func(r chi.Router) {
			r.Post("/upload-url", storageHandler.GenerateUploadURL)
			r.Post("/{id}/finalize", storageHandler.FinalizeUpload)
			r.Post("/{id}/abort", storageHandler.AbortUpload)
			r.Get("/{id}/download-url", storageHandler.GenerateDownloadURL)
		})

		r.Delete("/shares/{id}", storageHandler.RevokeShareLink)
	})

	// ----- /storage/v1/public/{token} (share-token required) -----
	tokenExtractor := func(req *http.Request) string { return chi.URLParam(req, "token") }
	r.Route("/storage/v1/public/{token}", func(r chi.Router) {
		r.Use(custommiddleware.ShareContext(useCase, tokenExtractor))

		r.Get("/", storageHandler.PublicShareInfo)
		r.Get("/tree", storageHandler.GetTree)
		r.Get("/folders/{id}/children", storageHandler.ListChildren)
		r.Get("/nodes/{id}", storageHandler.GetNode)
		r.Get("/files/{id}/download-url", storageHandler.GenerateDownloadURL)
		r.Patch("/nodes/{id}/rename", storageHandler.RenameNode)
		r.Delete("/nodes/{id}", storageHandler.SoftDeleteNode)
	})

	return r, rateLimiter
}
