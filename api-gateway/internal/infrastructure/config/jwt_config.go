package config

import "time"

type JWTConfig struct {
	JWKSUrl         string        `env:"JWT_JWKS_URL" env-default:"http://auth-service/.well-known/jwks.json"`
	RefreshInterval time.Duration `env:"JWT_JWKS_REFRESH_INTERVAL" env-default:"1h"`
	Issuer          string        `env:"JWT_ISSUER" env-default:"auth-service"`
	Audience        string        `env:"JWT_AUDIENCE" env-default:"cloud-storage"`
}
