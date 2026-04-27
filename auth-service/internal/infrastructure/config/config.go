package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env       string `env:"ENV" env-default:"local"`
	Server    ServerConfig
	Postgres  PostgresConfig
	JWT       JWTConfig
	Security  SecurityConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
}

func MustLoad() *Config {
	var cfg Config

	// 1. Попытка загрузить из .env файла
	// Мы игнорируем ошибку, если файла нет (например, в Docker-контейнере)
	_ = cleanenv.ReadConfig("deployments/.env", &cfg)

	// 2. Чтение переменных окружения (они приоритетнее файла)
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %s", err))
	}

	return &cfg
}
