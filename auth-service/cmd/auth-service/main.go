package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/usecase"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/config"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/database"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/security"
	httpserver "github.com/Grishanyaaaa/cloud-storage/auth-service/internal/presentation/http"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/presentation/http/handler"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/pkg/common/logger"
)

func main() {
	// 1. Загрузка конфигурации
	cfg := config.MustLoad()

	// 2. Инициализация логгера
	log := logger.SetupLogger(cfg.Env)
	log.Info("starting auth-service", slog.String("env", cfg.Env))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Подключение к БД с timeout
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	pool, err := database.NewPostgresPool(dbCtx, cfg.Postgres)
	if err != nil {
		log.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 4. Инициализация инфраструктуры (безопасность)
	passwordHasher := security.NewBcryptHasher(cfg.Security)
	tokenHasher := security.NewSHA256TokenHasher()
	tokenManager, err := security.NewJWTManager(cfg.JWT)
	if err != nil {
		log.Error("failed to init jwt manager", "error", err)
		os.Exit(1)
	}

	// 5. Инициализация репозиториев
	userRepo := database.NewUserRepository(pool)
	tokenRepo := database.NewRefreshTokenRepository(pool)
	auditRepo := database.NewAuditLogRepository(pool)

	// 6. Инициализация политики паролей
	passwordPolicy := security.NewPasswordPolicy(cfg.Security)

	// 7. Инициализация юзкейсов (Application layer)
	authUseCase := usecase.NewAuthService(
		userRepo,
		tokenRepo,
		auditRepo,
		passwordHasher,
		tokenManager,
		tokenHasher,
		passwordPolicy,
		log,
	)

	// 8. Инициализация презентации (HTTP)
	authHandler := handler.NewAuthHandler(authUseCase, tokenManager)
	router := httpserver.NewRouter(authHandler, cfg.CORS)
	srv := httpserver.NewServer(cfg.Server, router)

	// 9. Запуск сервера
	go func() {
		log.Info("server started", slog.Int("port", cfg.Server.Port))
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 10. Запуск периодической очистки истекших токенов (каждые 24 часа)
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()

	go func() {
		for {
			select {
			case <-cleanupTicker.C:
				// Create a new context with timeout for each cleanup operation
				// to avoid using the signal context which may be cancelled
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				deleted, err := authUseCase.CleanupExpiredTokens(cleanupCtx, log)
				cancel()
				if err != nil {
					log.Error("failed to cleanup expired tokens", "error", err)
				} else if deleted > 0 {
					log.Info("cleaned up expired tokens", slog.Int64("count", deleted))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	<-ctx.Done()
	log.Info("stopping server...")

	// Graceful Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	log.Info("server stopped gracefully")
}
