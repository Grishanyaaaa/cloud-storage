package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/usecase"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/config"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/database"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/id"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/objectstorage"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/security"
	infrattl "github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/ttl"
	httpserver "github.com/Grishanyaaaa/cloud-storage/storage-service/internal/presentation/http"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/presentation/http/handler"
	custommiddleware "github.com/Grishanyaaaa/cloud-storage/storage-service/internal/presentation/http/middleware"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/pkg/common/logger"
)

func main() {
	// 1. Configuration
	cfg := config.MustLoad()

	// 2. Logger
	log := logger.SetupLogger(strings.ToUpper(cfg.Env))
	log.Info("starting storage-service", slog.String("env", cfg.Env))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Postgres
	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()
	pool, err := database.NewPostgresPool(dbCtx, cfg.Postgres)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 4. Security adapters
	jwtParser, err := security.NewJWTParser(cfg.JWT)
	if err != nil {
		log.Error("failed to init jwt parser", "error", err)
		os.Exit(1)
	}
	tokenGen := security.NewShareTokenGenerator()

	// 5. Object storage (S3 / MinIO)
	s3Ctx, s3Cancel := context.WithTimeout(ctx, 10*time.Second)
	defer s3Cancel()
	objStorage, err := objectstorage.NewS3Client(s3Ctx, cfg.S3)
	if err != nil {
		log.Error("failed to init s3 client", "error", err)
		os.Exit(1)
	}

	// 6. Repositories + Tx manager
	nodeRepo := database.NewNodeRepository(pool)
	blobRepo := database.NewFileBlobRepository(pool)
	rootRepo := database.NewUserRootRepository(pool)
	shareRepo := database.NewShareRepository(pool)
	txManager := database.NewTransactionManager(pool)

	// 7. Adapters
	idGen := id.NewUUIDGenerator()
	ttlPolicy := infrattl.NewPolicy(cfg.TTL)

	// 8. Use case (composition root for application layer)
	storageUseCase := usecase.NewStorageService(
		nodeRepo,
		blobRepo,
		rootRepo,
		shareRepo,
		txManager,
		objStorage,
		ttlPolicy,
		idGen,
		tokenGen,
		cfg.Storage.PublicBaseURL,
		cfg.Storage.MaxFileSizeBytes,
		log,
	)

	// 9. Presentation
	storageHandler := handler.NewStorageHandler(storageUseCase, func(r *http.Request) *port.Actor {
		return custommiddleware.ActorFromContext(r.Context())
	})
	router, rateLimiter := httpserver.NewRouter(
		storageHandler,
		storageUseCase,
		jwtParser,
		cfg.CORS,
		cfg.Server,
	)
	srv := httpserver.NewServer(cfg.Server, router)

	// 10. Start HTTP server
	go func() {
		log.Info("server started", slog.Int("port", cfg.Server.Port))
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 11. Janitor goroutines
	pendingInterval := time.Duration(cfg.Storage.JanitorPendingUploadsIntervalSec) * time.Second
	if pendingInterval <= 0 {
		pendingInterval = 10 * time.Minute
	}
	sharesInterval := time.Duration(cfg.Storage.JanitorSharesIntervalSec) * time.Second
	if sharesInterval <= 0 {
		sharesInterval = 10 * time.Minute
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go runJanitor(
		ctx, &wg, log, "pending-uploads", pendingInterval,
		storageUseCase.JanitorExpirePendingUploads,
	)
	go runJanitor(
		ctx, &wg, log, "shares", sharesInterval,
		storageUseCase.JanitorExpireShares,
	)

	<-ctx.Done()
	log.Info("stopping server...")

	rateLimiter.Stop()
	wg.Wait()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Stop(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}
	log.Info("server stopped")
}

func runJanitor(
	ctx context.Context,
	wg *sync.WaitGroup,
	log *slog.Logger,
	name string,
	interval time.Duration,
	tick func(context.Context) (int64, error),
) {
	defer wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			affected, err := tick(cctx)
			cancel()
			if err != nil {
				log.Error("janitor tick failed", "name", name, "error", err)
			} else if affected > 0 {
				log.Info("janitor tick", "name", name, slog.Int64("affected", affected))
			}
		case <-ctx.Done():
			return
		}
	}
}
