package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/config"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(cfg config.ServerConfig, h http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      h,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
