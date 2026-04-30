package config

type AuthServiceConfig struct {
	URL string `env:"AUTH_SERVICE_URL" env-default:"http://auth-service"`
}
