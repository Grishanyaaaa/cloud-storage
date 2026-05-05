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

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/usecase"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/config"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/database"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/id"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/llm"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/security"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/storageclient"
	httpserver "github.com/Grishanyaaaa/cloud-storage/ai-service/internal/presentation/http"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/presentation/http/handler"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/pkg/common/logger"
)

func main() {
	// 1. Configuration
	cfg := config.MustLoad()

	// 2. Logger
	log := logger.SetupLogger(strings.ToUpper(cfg.Env))
	log.Info("starting ai-service", slog.String("env", cfg.Env))

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

	// 4. Security adapters (JWT verify-only)
	jwtParser, err := security.NewJWTParser(cfg.JWT)
	if err != nil {
		log.Error("failed to init jwt parser", "error", err)
		os.Exit(1)
	}

	// 5. Adapters: LLM, storage-service, repository, id generator
	llmClient := llm.NewYandexGPTClient(cfg.YandexGPT)
	storageClient := storageclient.NewStorageClient(cfg.StorageService)
	cmdRepo := database.NewPostgresAiCommandRepository(pool)
	txManager := database.NewTransactionManager(pool)
	idGen := id.NewUUIDGenerator()

	// 6. Use case (composition root for application layer)
	aiUseCase := usecase.NewAIService(usecase.AIServiceConfig{
		CmdRepo:       cmdRepo,
		TxManager:     txManager,
		Storage:       storageClient,
		LLM:           llmClient,
		IDs:           idGen,
		MaxInputChars: cfg.AI.MaxInputChars,
		PlanTTL:       cfg.AI.PlanTTL,
		MaxRetries:    cfg.AI.MaxLLMRetries,
		LLMModel:      cfg.YandexGPT.EffectiveModelURI(),
		Logger:        log,
	})

	// 7. Presentation
	aiHandler := handler.NewAIHandler(aiUseCase, httpserver.DefaultActorExtractor)
	router := httpserver.NewRouter(aiHandler, jwtParser, cfg.CORS)
	srv := httpserver.NewServer(cfg.Server, router)

	// 8. Start HTTP server
	go func() {
		log.Info("server started", slog.Int("port", cfg.Server.Port))
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 9. Janitor goroutine: expire pending plans
	janitorInterval := cfg.AI.JanitorPlansInterval
	if janitorInterval <= 0 {
		janitorInterval = 60 * time.Second
	}
	janitor := usecase.NewJanitorExpirePendingPlans(cmdRepo, cfg.AI.JanitorBatchSize, log)

	var wg sync.WaitGroup
	wg.Add(1)
	go runJanitor(ctx, &wg, log, "pending-plans", janitorInterval, janitor.Run)

	// 10. Wait for shutdown signal
	<-ctx.Done()
	log.Info("stopping server...")

	wg.Wait()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Stop(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}
	log.Info("server stopped")
}

// runJanitor invokes `tick` every `interval` until ctx is done.
// Tick failures are logged and ignored — the next tick gets a fresh context.
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
