package config

type CORSConfig struct {
	AllowOrigins     string `env:"CORS_ALLOW_ORIGINS" env-default:"*"`
	AllowMethods     string `env:"CORS_ALLOW_METHODS" env-default:"GET,POST,PUT,DELETE,OPTIONS"`
	AllowHeaders     string `env:"CORS_ALLOW_HEADERS" env-default:"Content-Type,Authorization"`
	AllowCredentials bool   `env:"CORS_ALLOW_CREDENTIALS" env-default:"false"`
	MaxAge           int    `env:"CORS_MAX_AGE" env-default:"86400"`
}
