package config

type StorageServiceConfig struct {
	URL string `env:"STORAGE_SERVICE_URL" env-default:"http://storage-service"`
}
