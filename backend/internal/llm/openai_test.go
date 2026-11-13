package llm

import (
	"context"
	"encoding/json"
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
		var in map[string]any
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if in["model"] != "test-model" {
			t.Errorf("model = %v", in["model"])
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
