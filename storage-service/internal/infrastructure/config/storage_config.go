package config

// StorageConfig holds general storage-service tunables.
type StorageConfig struct {
	// MaxFileSizeBytes — единый верхний предел на размер файла.
	// 5 GiB = 5_368_709_120 bytes (по ТЗ).
	MaxFileSizeBytes int64 `env:"STORAGE_MAX_FILE_SIZE_BYTES" env-default:"5368709120"`

	// PublicBaseURL — публичный URL сервиса для построения share-ссылок
	// (например https://files.example.com). Без trailing slash.
	PublicBaseURL string `env:"STORAGE_PUBLIC_BASE_URL" env-default:""`

	// JanitorPendingUploadsInterval — частота уборки зависших pending blob'ов.
	JanitorPendingUploadsIntervalSec int `env:"STORAGE_JANITOR_PENDING_INTERVAL_SEC" env-default:"600"`

	// JanitorSharesInterval — частота уборки истёкших shares.
	JanitorSharesIntervalSec int `env:"STORAGE_JANITOR_SHARES_INTERVAL_SEC" env-default:"600"`

	// JanitorBatchSize — макс. количество строк, обрабатываемых за один тик janitor'а.
	JanitorBatchSize int `env:"STORAGE_JANITOR_BATCH_SIZE" env-default:"500"`
}
