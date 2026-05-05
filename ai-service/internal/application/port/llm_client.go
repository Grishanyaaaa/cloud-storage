package port

import "context"

// LLMRequest is a single, non-streamed LLM completion request.
//
//	SystemPrompt — high-level instruction in Russian (rules, format, examples)
//	UserMessage  — user input + serialized tree context
//	JSONSchema   — optional JSON Schema for structured output (Yandex GPT supports this)
//	Temperature  — 0.0..1.0; lower = more deterministic
//	MaxTokens    — hard cap on response tokens
type LLMRequest struct {
	SystemPrompt string
	UserMessage  string
	JSONSchema   map[string]any
	Temperature  float64
	MaxTokens    int
}

// LLMResponse is the raw, untyped response from the provider.
//
//	Text       — model's answer (expected to be a JSON string when JSONSchema was set)
//	Model      — actual model URI / version reported by provider
//	TokensIn   — prompt token count (best-effort, from usage section)
//	TokensOut  — completion token count
type LLMResponse struct {
	Text      string
	Model     string
	TokensIn  int
	TokensOut int
}

// LLMClient sends synchronous completion requests to a remote LLM provider.
// On transport/HTTP failures it returns domainerr.ErrLLMUnavailable.
// On any provider-side error (4xx/5xx) it also returns ErrLLMUnavailable —
// payload-level validity is the caller's responsibility.
type LLMClient interface {
	Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
}
