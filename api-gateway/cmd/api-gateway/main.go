package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/infrastructure/client"
	"github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/infrastructure/config"
	httpserver "github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/presentation/http"
	"github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/presentation/http/handler"
	"github.com/Grishanyaaaa/cloud-storage/api-gateway/pkg/common/logger"
)

func main() {
	// 1. Загрузка конфигурации
	cfg := config.MustLoad()

	// 2. Инициализация логгера
	log := logger.SetupLogger(strings.ToUpper(cfg.Env))
	log.Info("starting api-gateway", slog.String("env", cfg.Env))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Инициализация JWKS клиента
	jwksClient := client.NewJWKSClient(cfg.JWT)

	// 4. Первоначальная загрузка JWKS
	log.Info("fetching JWKS from auth-service", slog.String("url", cfg.JWT.JWKSUrl))
	fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := jwksClient.FetchKeys(fetchCtx); err != nil {
		log.Error("failed to fetch initial JWKS", "error", err)
		fetchCancel()
		os.Exit(1)
	}
	fetchCancel()
	log.Info("JWKS fetched successfully")

	// 5. Запуск фонового обновления JWKS
	go jwksClient.StartBackgroundRefresh(ctx)

	// 6. Инициализация handlers
	proxyHandler := handler.NewProxyHandler(cfg.AuthService.URL, cfg.StorageService.URL, cfg.AIService.URL)
	healthHandler := handler.NewHealthHandler(cfg.AuthService.URL, cfg.StorageService.URL, cfg.AIService.URL)

	// 7. Инициализация роутера
	router, rateLimiter := httpserver.NewRouter(proxyHandler, healthHandler, jwksClient, cfg)
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

	// Stop rate limiter cleanup goroutine
	rateLimiter.Stop()

	// Graceful Shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	log.Info("server stopped")
}
