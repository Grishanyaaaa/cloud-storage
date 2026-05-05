// Package llm provides adapters to remote LLM providers.
//
// YandexGPTClient implements port.LLMClient against Yandex Cloud's
// Foundation Models API.
//
//	Endpoint:    https://llm.api.cloud.yandex.net/foundationModels/v1/completion
//	Auth:        header "Authorization: Api-Key <key>" + "x-folder-id: <folder>"
//	JSON Schema: completion option `responseFormat.jsonSchema.schema`
//	Token usage: parsed from `result.usage.{inputTextTokens, completionTokens}`
//	Output:      text in `result.alternatives[0].message.text`
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/config"
)

// Compile-time check: YandexGPTClient implements port.LLMClient
var _ port.LLMClient = (*YandexGPTClient)(nil)

// YandexGPTClient is a synchronous, non-streaming HTTP adapter over the
// Yandex Foundation Models REST API.
type YandexGPTClient struct {
	cfg    config.YandexGPTConfig
	http   *http.Client
}

// NewYandexGPTClient builds a YandexGPTClient with a dedicated http.Client
// (timeout = cfg.Timeout). Caller does not need to close anything.
func NewYandexGPTClient(cfg config.YandexGPTConfig) *YandexGPTClient {
	return &YandexGPTClient{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// ---------- request shape (Yandex API) ----------

type yandexCompletionRequest struct {
	ModelURI          string                       `json:"modelUri"`
	CompletionOptions yandexCompletionOptions      `json:"completionOptions"`
	Messages          []yandexMessage              `json:"messages"`
	JSONSchema        *yandexJSONSchemaWrapper     `json:"jsonSchema,omitempty"`
}

type yandexCompletionOptions struct {
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature"`
	MaxTokens   string  `json:"maxTokens"`
}

type yandexMessage struct {
	Role string `json:"role"` // "system" | "user" | "assistant"
	Text string `json:"text"`
}

// yandexJSONSchemaWrapper is the preferred way to enforce structured output
// in the Foundation Models REST API:
//
//	"jsonSchema": { "schema": { ... } }
type yandexJSONSchemaWrapper struct {
	Schema map[string]any `json:"schema"`
}

// ---------- response shape (Yandex API) ----------

type yandexCompletionResponse struct {
	Result yandexResult `json:"result"`
	Error  *yandexError `json:"error,omitempty"`
}

type yandexResult struct {
	Alternatives []yandexAlternative `json:"alternatives"`
	Usage        yandexUsage         `json:"usage"`
	ModelVersion string              `json:"modelVersion"`
}

type yandexAlternative struct {
	Message yandexMessage `json:"message"`
	Status  string        `json:"status"`
}

type yandexUsage struct {
	InputTextTokens  string `json:"inputTextTokens"`
	CompletionTokens string `json:"completionTokens"`
	TotalTokens      string `json:"totalTokens"`
}

type yandexError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// ---------- LLMClient implementation ----------

// Complete sends a single completion request and returns the raw response.
func (c *YandexGPTClient) Complete(ctx context.Context, req port.LLMRequest) (port.LLMResponse, error) {
	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.cfg.Temperature
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.cfg.MaxTokens
	}

	body := yandexCompletionRequest{
		ModelURI: c.cfg.EffectiveModelURI(),
		CompletionOptions: yandexCompletionOptions{
			Stream:      false,
			Temperature: temperature,
			MaxTokens:   strconv.Itoa(maxTokens),
		},
		Messages: []yandexMessage{
			{Role: "system", Text: req.SystemPrompt},
			{Role: "user", Text: req.UserMessage},
		},
	}
	if req.JSONSchema != nil {
		body.JSONSchema = &yandexJSONSchemaWrapper{Schema: req.JSONSchema}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return port.LLMResponse{}, fmt.Errorf("marshal yandex completion request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return port.LLMResponse{}, fmt.Errorf("build yandex request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Api-Key "+c.cfg.APIKey)
	httpReq.Header.Set("x-folder-id", c.cfg.FolderID)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return port.LLMResponse{}, domainerr.New(
			domainerr.CodeLLMUnavailable,
			"yandex gpt request failed",
			err,
		)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return port.LLMResponse{}, domainerr.New(
			domainerr.CodeLLMUnavailable,
			"read yandex gpt response body",
			err,
		)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return port.LLMResponse{}, domainerr.New(
			domainerr.CodeLLMUnavailable,
			fmt.Sprintf("yandex gpt returned status %d: %s", resp.StatusCode, truncate(string(respBody), 256)),
			nil,
		)
	}

	var parsed yandexCompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return port.LLMResponse{}, domainerr.New(
			domainerr.CodeLLMInvalidResponse,
			"yandex gpt returned malformed JSON envelope",
			err,
		)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return port.LLMResponse{}, domainerr.New(
			domainerr.CodeLLMUnavailable,
			fmt.Sprintf("yandex gpt error: %s (code=%d)", parsed.Error.Message, parsed.Error.Code),
			nil,
		)
	}
	if len(parsed.Result.Alternatives) == 0 {
		return port.LLMResponse{}, domainerr.ErrLLMInvalidResponse
	}
	text := parsed.Result.Alternatives[0].Message.Text

	return port.LLMResponse{
		Text:      text,
		Model:     parsed.Result.ModelVersion,
		TokensIn:  parseIntOrZero(parsed.Result.Usage.InputTextTokens),
		TokensOut: parseIntOrZero(parsed.Result.Usage.CompletionTokens),
	}, nil
}

// truncate returns at most n runes of s, suffixed by "…" when truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func parseIntOrZero(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

