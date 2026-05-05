package config

type AIServiceConfig struct {
	URL string `env:"AI_SERVICE_URL" env-default:"http://ai-service"`
}
