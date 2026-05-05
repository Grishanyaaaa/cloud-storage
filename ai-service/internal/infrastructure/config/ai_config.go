package config

import "time"

// AIConfig holds general ai-service tunables.
type AIConfig struct {
	// MaxInputChars — верхняя граница длины пользовательского input'а
	// (защита от случайных гигантских команд; rate-limit'ов в MVP нет).
	MaxInputChars int `env:"AI_MAX_INPUT_CHARS" env-default:"2000"`

	// PlanTTL — сколько живёт plan в статусе awaiting_confirmation
	// до того, как janitor его пометит expired.
	PlanTTL time.Duration `env:"AI_PLAN_TTL" env-default:"5m"`

	// MaxLLMRetries — макс. количество дополнительных попыток вызова LLM
	// при невалидном (неразбираемом / невалидируемом) ответе. 1 → итого 2 попытки.
	MaxLLMRetries int `env:"AI_MAX_LLM_RETRIES" env-default:"1"`

	// JanitorPlansInterval — частота работы janitor'а expired plans.
	JanitorPlansInterval time.Duration `env:"AI_JANITOR_PLANS_INTERVAL" env-default:"60s"`

	// JanitorBatchSize — макс. количество строк, обрабатываемых за один тик janitor'а.
	JanitorBatchSize int `env:"AI_JANITOR_BATCH_SIZE" env-default:"500"`
}
