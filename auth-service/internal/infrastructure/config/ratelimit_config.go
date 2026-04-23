package config

import "time"

type RateLimitConfig struct {
	RequestsPerSecond float64       `env:"RATE_LIMIT_RPS" env-default:"10"`
	Burst             int           `env:"RATE_LIMIT_BURST" env-default:"20"`
	CleanupInterval   time.Duration `env:"RATE_LIMIT_CLEANUP_INTERVAL" env-default:"5m"`
}
