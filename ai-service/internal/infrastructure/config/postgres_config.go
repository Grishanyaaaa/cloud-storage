package config

import (
	"fmt"
	"net/url"
	"time"
)

type PostgresConfig struct {
	Host            string        `env:"POSTGRES_HOST" env-default:"localhost"`
	Port            int           `env:"POSTGRES_PORT" env-default:"5432"`
	User            string        `env:"POSTGRES_USER" env-required:"true"`
	Password        string        `env:"POSTGRES_PASSWORD" env-required:"true"`
	Database        string        `env:"POSTGRES_DB" env-required:"true"`
	SSLMode         string        `env:"POSTGRES_SSLMODE" env-default:"disable"`
	MaxOpenConns    int           `env:"POSTGRES_MAX_OPEN_CONNS" env-default:"10"`
	MaxIdleConns    int           `env:"POSTGRES_MAX_IDLE_CONNS" env-default:"5"`
	ConnMaxLifetime time.Duration `env:"POSTGRES_CONN_MAX_LIFETIME" env-default:"5m"`
	ConnMaxIdleTime time.Duration `env:"POSTGRES_CONN_MAX_IDLE_TIME" env-default:"1m"`
}

func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, url.QueryEscape(c.Password), c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// String returns a safe representation of PostgresConfig with masked password.
// Used for logging to prevent password leakage.
func (c PostgresConfig) String() string {
	return fmt.Sprintf(
		"PostgresConfig{Host:%s Port:%d User:%s Database:%s SSLMode:%s MaxOpenConns:%d MaxIdleConns:%d}",
		c.Host, c.Port, c.User, c.Database, c.SSLMode, c.MaxOpenConns, c.MaxIdleConns,
	)
}
