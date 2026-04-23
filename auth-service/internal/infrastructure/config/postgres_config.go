package config

import (
	"fmt"
	"time"
)

type PostgresConfig struct {
	Host            string        `env:"POSTGRES_HOST" env-default:"localhost"`
	Port            int           `env:"POSTGRES_PORT" env-default:"5432"`
	User            string        `env:"POSTGRES_USER" env-required:"true"`
	Password        string        `env:"POSTGRES_PASSWORD" env-required:"true"`
	Database        string        `env:"POSTGRES_DB" env-required:"true"`
	SSLMode         string        `env:"POSTGRES_SSL_MODE" env-default:"disable"`
	MaxOpenConns    int           `env:"POSTGRES_MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns    int           `env:"POSTGRES_MAX_IDLE_CONNS" env-default:"10"`
	ConnMaxLifetime time.Duration `env:"POSTGRES_CONN_MAX_LIFETIME" env-default:"5m"`
	ConnMaxIdleTime time.Duration `env:"POSTGRES_CONN_MAX_IDLE_TIME" env-default:"1m"`
}

func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}
