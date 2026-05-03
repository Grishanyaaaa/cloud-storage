package config

import "time"

// TTLConfig describes parameters of the size-based pre-signed URL TTL policy.
//
// Algorithm (см. blueprint §13.1):
//
//	upload TTL   = clamp(BaseUpload + sizeMiB * PerMiBUpload, MinUpload, MaxUpload)
//	download TTL = clamp(BaseDownload + sizeMiB * PerMiBDownload, MinDownload, MaxDownload)
type TTLConfig struct {
	BaseUpload     time.Duration `env:"TTL_BASE_UPLOAD" env-default:"5m"`
	PerMiBUpload   time.Duration `env:"TTL_PER_MIB_UPLOAD" env-default:"5s"`
	MinUpload      time.Duration `env:"TTL_MIN_UPLOAD" env-default:"5m"`
	MaxUpload      time.Duration `env:"TTL_MAX_UPLOAD" env-default:"6h"`
	BaseDownload   time.Duration `env:"TTL_BASE_DOWNLOAD" env-default:"2m"`
	PerMiBDownload time.Duration `env:"TTL_PER_MIB_DOWNLOAD" env-default:"2s"`
	MinDownload    time.Duration `env:"TTL_MIN_DOWNLOAD" env-default:"2m"`
	MaxDownload    time.Duration `env:"TTL_MAX_DOWNLOAD" env-default:"2h"`
}
