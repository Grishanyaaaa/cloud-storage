package config

import "time"

type JWTConfig struct {
	PrivateKey      string        `env:"JWT_PRIVATE_KEY" env-required:"true"` // Base64 encoded 32-byte seed
	PublicKey       string        `env:"JWT_PUBLIC_KEY" env-required:"true"`  // Base64 encoded 32-byte public key
	AccessTokenTTL  time.Duration `env:"JWT_ACCESS_TOKEN_TTL" env-default:"15m"`
	RefreshTokenTTL time.Duration `env:"JWT_REFRESH_TOKEN_TTL" env-default:"720h"`
	Issuer          string        `env:"JWT_ISSUER" env-default:"auth-service"`
	Audience        string        `env:"JWT_AUDIENCE" env-default:"cloud-storage"`
}
