// Package llm isolates AI provider access behind a thin abstraction
// (spec §6/§10.2): the rest of the codebase depends on Client, never on an
// SDK or a specific provider. The only implementation is an OpenAI-compatible
// HTTP client (docs/adr/0005), which covers OpenAI, Groq, Together,
// Fireworks, Ollama and any other /v1-compatible endpoint.
package llm

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Config carries everything needed to reach a provider. It travels with each
// call (instead of living inside the client) because the values come from the
// encrypted settings store and may change at runtime — BYOK, no restarts.
type Config struct {
	BaseURL string // e.g. https://api.openai.com/v1
	APIKey  string
	Model   string // e.g. gpt-4o-mini, llama-3.3-70b-versatile
}

// Message is one chat turn.
type Message struct {
	Role    string // 'system' | 'user' | 'assistant'
	Content string
}

// CompletionRequest is a minimal chat completion request.
type CompletionRequest struct {
	Messages    []Message
	MaxTokens   int // 0 = provider default
	Temperature float64
}

// Client is the provider boundary used by CP4 (tailoring) and CP5 (scoring).
type Client interface {
	// Ping verifies that the provider is reachable and the credentials/model
	// are accepted. It returns a legible error on any failure.
	Ping(ctx context.Context, cfg Config) error
	// Complete runs a chat completion and returns the assistant's text.
	Complete(ctx context.Context, cfg Config, req CompletionRequest) (string, error)
}

// Defaults used when the user leaves fields empty in settings.
const (
	DefaultBaseURL = "https://api.openai.com/v1"
	DefaultModel   = "gpt-4o-mini"
)

// Normalize fills empty fields with defaults and trims the base URL.
func (c Config) Normalize() Config {
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.Model == "" {
		c.Model = DefaultModel
	}
	return c
}

// requestTimeout bounds each provider call; LLM answers are slow but never
// should hang a request indefinitely.
const requestTimeout = 120 * time.Second

// WithTimeout wraps ctx with the provider call budget.
func WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, requestTimeout)
}

// ErrNotConfigured is returned when a call is attempted with an empty API key.
var ErrNotConfigured = fmt.Errorf("llm: provider not configured (missing api key)")
