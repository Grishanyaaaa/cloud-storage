// Package llm provides adapters to remote LLM providers.
//
// YandexGPTClient implements port.LLMClient against Yandex Cloud's
// Foundation Models API v1/responses.
//
//	Endpoint:    https://ai.api.cloud.yandex.net/v1/responses
//	Auth:        header "Authorization: Api-Key <key>" + "OpenAI-Project: <folder>"
//	JSON Schema: response_format.json_schema
//	Output:      text in `output[0].content[0].text`
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

// ---------- request shape (Yandex API v1/responses) ----------

type yandexCompletionRequest struct {
	Model           string             `json:"model"`
	Temperature     float64            `json:"temperature"`
	Instructions    string             `json:"instructions"`
	Input           string             `json:"input"`
	MaxOutputTokens int                `json:"max_output_tokens"`
	ResponseFormat  *yandexResponseFormat `json:"response_format,omitempty"`
}

type yandexResponseFormat struct {
	Type       string         `json:"type"` // "json_schema"
	JSONSchema map[string]any `json:"json_schema,omitempty"`
}

// ---------- response shape (Yandex API v1/responses) ----------

type yandexCompletionResponse struct {
	Output []yandexOutput `json:"output"`
	Error  *yandexError   `json:"error,omitempty"`
}

type yandexOutput struct {
	Content []yandexContent `json:"content"`
}

type yandexContent struct {
	Text string `json:"text"`
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

	// Combine system prompt and user message into instructions and input
	instructions := req.SystemPrompt
	input := req.UserMessage

	body := yandexCompletionRequest{
		Model:           c.cfg.EffectiveModelURI(),
		Temperature:     temperature,
		Instructions:    instructions,
		Input:           input,
		MaxOutputTokens: maxTokens,
	}
	if req.JSONSchema != nil {
		body.ResponseFormat = &yandexResponseFormat{
			Type:       "json_schema",
			JSONSchema: req.JSONSchema,
		}
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
	httpReq.Header.Set("OpenAI-Project", c.cfg.FolderID)

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

	// Debug: log raw response
	fmt.Printf("[DEBUG] Yandex GPT raw response: %s\n", string(respBody))

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
	if len(parsed.Output) == 0 || len(parsed.Output[0].Content) == 0 {
		return port.LLMResponse{}, domainerr.ErrLLMInvalidResponse
	}
	text := parsed.Output[0].Content[0].Text

	return port.LLMResponse{
		Text:      text,
		Model:     c.cfg.Model,
		TokensIn:  0, // v1/responses API doesn't return token usage
		TokensOut: 0,
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

