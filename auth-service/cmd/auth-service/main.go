package main

import (
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/config"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/pkg/common/logger"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Env)
	log.Info("Hello World")
}
