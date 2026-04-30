package config

import "time"

type ServerConfig struct {
	Host            string        `env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port            int           `env:"SERVER_PORT" env-default:"8080"`
	ReadTimeout     time.Duration `env:"SERVER_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout    time.Duration `env:"SERVER_WRITE_TIMEOUT" env-default:"15s"`
	IdleTimeout     time.Duration `env:"SERVER_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" env-default:"30s"`
}
