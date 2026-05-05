package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env            string `env:"ENV" env-default:"local"`
	Server         ServerConfig
	Postgres       PostgresConfig
	JWT            JWTConfig
	StorageService StorageServiceConfig
	YandexGPT      YandexGPTConfig
	AI             AIConfig
	CORS           CORSConfig
}

func MustLoad() *Config {
	var cfg Config

	// Ignore error if .env file doesn't exist (e.g., in Docker with env vars)
	_ = cleanenv.ReadConfig("deployments/.env", &cfg)

	// 2. Чтение переменных окружения (они приоритетнее файла)
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %s", err))
	}

	return &cfg
}
