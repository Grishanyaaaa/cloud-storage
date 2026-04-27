package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	// 3. Подключение к БД
	pool, err := database.NewPostgresPool(ctx, cfg.Postgres)
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

	// 6. Инициализация юзкейсов (Application layer)
	authUseCase := usecase.NewAuthService(
		userRepo,
		tokenRepo,
		auditRepo,
		passwordHasher,
		tokenManager,
		tokenHasher,
	)

	// 7. Инициализация презентации (HTTP)
	authHandler := handler.NewAuthHandler(authUseCase, tokenManager)
	router := httpserver.NewRouter(authHandler)
	srv := httpserver.NewServer(cfg.Server, router)

	// 8. Запуск сервера
	go func() {
		log.Info("server started", slog.Int("port", cfg.Server.Port))
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
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
