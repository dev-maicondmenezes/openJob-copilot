package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPingSuccess(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("ping path = %s, want /models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("authorization = %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer provider.Close()

	client := NewOpenAIClient()
	if err := client.Ping(context.Background(), Config{
		BaseURL: provider.URL, APIKey: "sk-test", Model: "m",
	}); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestPingSurfacesProviderError(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Incorrect API key provided"}}`))
	}))
	defer provider.Close()

	client := NewOpenAIClient()
	err := client.Ping(context.Background(), Config{BaseURL: provider.URL, APIKey: "bad"})
	if err == nil || !strings.Contains(err.Error(), "Incorrect API key provided") {
		t.Fatalf("err = %v, want provider message", err)
	}
}

// Security regression: the provider error body is echoed verbatim — a hostile
// or buggy provider may include the caller's credentials in that body. The
// key must be masked before the error leaves the llm package (it goes into
// server logs and HTTP responses).
func TestErrorsNeverContainTheAPIKey(t *testing.T) {
	secret := "sk-leak-me-not-1234567890"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid key: ` + secret + `"}`))
	}))
	defer provider.Close()

	client := NewOpenAIClient()
	cfg := Config{BaseURL: provider.URL, APIKey: secret, Model: "m"}

	err := client.Ping(context.Background(), cfg)
	if err == nil {
		t.Fatal("want error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("Ping error leaked the API key: %v", err)
	}
	if !strings.Contains(err.Error(), "***") {
		t.Errorf("Ping error did not mask the key: %v", err)
	}

	// errors.Is survives the redaction wrapper.
	cfgEmpty := Config{BaseURL: provider.URL, APIKey: ""}
	if e := client.Ping(context.Background(), cfgEmpty); !errors.Is(e, ErrNotConfigured) {
		t.Errorf("errors.Is(ErrNotConfigured) broken by redaction: %v", e)
	}

	_, err = client.Complete(context.Background(), cfg,
		CompletionRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Errorf("Complete error leaked the API key: %v", err)
	}

	// Transport-level errors too (e.g. timeouts mention the URL, never the
	// key — but the same scrub must apply).
	unreachable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	unreachable.Close()
	_, err = client.Complete(context.Background(), Config{BaseURL: unreachable.URL, APIKey: secret, Model: "m"},
		CompletionRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Errorf("transport error leaked the API key: %v", err)
	}
}

func TestPingNotConfigured(t *testing.T) {
	client := NewOpenAIClient()
	if err := client.Ping(context.Background(), Config{BaseURL: "https://x", APIKey: ""}); err != ErrNotConfigured {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestCompleteHappyPath(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		// Contract test: the wire format MUST use lowercase OpenAI keys.
		// (Regression guard: Go field names "Role"/"Content" without json
		// tags made Gemini reject the request with 400 INVALID_ARGUMENT.)
		var in struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &in); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if len(in.Messages) != 1 || in.Messages[0].Role != "user" || in.Messages[0].Content == "" {
			t.Errorf("wire format wrong: messages = %s", raw)
		}
		if strings.Contains(string(raw), "\"Role\"") || strings.Contains(string(raw), "\"Content\"") {
			t.Errorf("Go field names leaked to the wire: %s", raw)
		}
		if in.Model != "test-model" {
			t.Errorf("model = %v", in.Model)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "olá, mundo"}},
			},
		})
	}))
	defer provider.Close()

	client := NewOpenAIClient()
	out, err := client.Complete(context.Background(),
		Config{BaseURL: provider.URL, APIKey: "sk-test", Model: "test-model"},
		CompletionRequest{Messages: []Message{{Role: "user", Content: "hi"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if out != "olá, mundo" {
		t.Errorf("out = %q", out)
	}
}

func TestConfigNormalize(t *testing.T) {
	got := Config{BaseURL: "https://api.openai.com/v1/", Model: ""}.Normalize()
	if got.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %q, want trailing slash trimmed", got.BaseURL)
	}
	if got.Model != DefaultModel {
		t.Errorf("Model = %q, want default %q", got.Model, DefaultModel)
	}
}
