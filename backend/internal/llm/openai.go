package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIClient is a thin OpenAI-compatible client over net/http — no SDK, so
// every provider quirk stays visible and fixable in one file (docs/adr/0005).
type OpenAIClient struct {
	// HTTP is used for all calls; nil means http.DefaultClient.
	HTTP *http.Client
}

var _ Client = (*OpenAIClient)(nil)

// NewOpenAIClient returns a Client ready for use.
func NewOpenAIClient() *OpenAIClient { return &OpenAIClient{} }

func (c *OpenAIClient) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// Ping hits GET {base}/models with the configured key. A 200 means the
// endpoint is reachable and the key accepted — enough for "Testar conexão".
func (c *OpenAIClient) Ping(ctx context.Context, cfg Config) error {
	cfg = cfg.Normalize()
	if cfg.APIKey == "" {
		return ErrNotConfigured
	}

	ctx, cancel := WithTimeout(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.BaseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("llm: ping request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("llm: ping %s: %w", cfg.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("llm: ping %s: %s", cfg.BaseURL, describeError(resp))
}

// Complete runs POST /chat/completions and returns the first choice text.
func (c *OpenAIClient) Complete(ctx context.Context, cfg Config, creq CompletionRequest) (string, error) {
	cfg = cfg.Normalize()
	if cfg.APIKey == "" {
		return "", ErrNotConfigured
	}
	if len(creq.Messages) == 0 {
		return "", fmt.Errorf("llm: completion with no messages")
	}

	body := map[string]any{
		"model":       cfg.Model,
		"messages":    creq.Messages,
		"temperature": creq.Temperature,
	}
	if creq.MaxTokens > 0 {
		body["max_tokens"] = creq.MaxTokens
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("llm: encode request: %w", err)
	}

	ctx, cancel := WithTimeout(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("llm: completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: completion %s: %w", cfg.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: completion %s: %s", cfg.BaseURL, describeError(resp))
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("llm: decode completion: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("llm: completion returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}

// describeError extracts a readable message from a non-2xx response. Most
// OpenAI-compatible providers answer {"error": {"message": "..."}}.
func describeError(resp *http.Response) string {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if err != nil {
		return fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &body) == nil && body.Error.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(body.Error.Message))
	}
	if msg := strings.TrimSpace(string(raw)); msg != "" {
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, msg)
	}
	return fmt.Sprintf("HTTP %d", resp.StatusCode)
}
