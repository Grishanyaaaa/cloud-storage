package config

import "time"

// YandexGPTConfig configures the Yandex Cloud Foundation Models REST client.
// Endpoint: https://llm.api.cloud.yandex.net/foundationModels/v1/completion
// Authentication: header "Authorization: Api-Key <ApiKey>" + "x-folder-id: <FolderID>".
type YandexGPTConfig struct {
	APIKey      string        `env:"YANDEX_GPT_API_KEY" env-required:"true"`
	FolderID    string        `env:"YANDEX_GPT_FOLDER_ID" env-required:"true"`
	ModelURI    string        `env:"YANDEX_GPT_MODEL_URI" env-default:""` // если пусто, собираем из FolderID + Model
	Model       string        `env:"YANDEX_GPT_MODEL" env-default:"yandexgpt/latest"`
	Endpoint    string        `env:"YANDEX_GPT_ENDPOINT" env-default:"https://llm.api.cloud.yandex.net/foundationModels/v1/completion"`
	Timeout     time.Duration `env:"YANDEX_GPT_TIMEOUT" env-default:"60s"`
	Temperature float64       `env:"YANDEX_GPT_TEMPERATURE" env-default:"0.1"`
	MaxTokens   int           `env:"YANDEX_GPT_MAX_TOKENS" env-default:"2000"`
}

// EffectiveModelURI returns the value passed to the Yandex API as `modelUri`.
// If the user provided an explicit ModelURI we use it as-is; otherwise we build
// `gpt://<FolderID>/<Model>` (the canonical form expected by Yandex).
func (c YandexGPTConfig) EffectiveModelURI() string {
	if c.ModelURI != "" {
		return c.ModelURI
	}
	return "gpt://" + c.FolderID + "/" + c.Model
}
