package config

import "time"

// StorageServiceConfig configures the HTTP client to storage-service.
// JWT is propagated from the inbound request — there is no service-to-service token.
type StorageServiceConfig struct {
	BaseURL       string        `env:"STORAGE_SERVICE_BASE_URL" env-default:"http://storage-service:8082"`
	Timeout       time.Duration `env:"STORAGE_SERVICE_TIMEOUT" env-default:"15s"`
	TreeMaxDepth  int           `env:"STORAGE_SERVICE_TREE_MAX_DEPTH" env-default:"10"`
	TreeMaxNodes  int           `env:"STORAGE_SERVICE_TREE_MAX_NODES" env-default:"500"`
}
