package config

import "time"

type JWTConfig struct {
	PrivateKeyPath  string        `env:"JWT_PRIVATE_KEY_PATH" env-required:"true"`
	PublicKeyPath   string        `env:"JWT_PUBLIC_KEY_PATH" env-required:"true"`
	AccessTokenTTL  time.Duration `env:"JWT_ACCESS_TOKEN_TTL" env-default:"15m"`
	RefreshTokenTTL time.Duration `env:"JWT_REFRESH_TOKEN_TTL" env-default:"720h"`
	Issuer          string        `env:"JWT_ISSUER" env-default:"auth-service"`
	Audience        string        `env:"JWT_AUDIENCE" env-default:"cloud-storage"`
}
