package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/llm"
)

// postJSON is a tiny helper for the settings endpoints.
func postJSON(t *testing.T, url string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out := map[string]any{}
	if resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil && resp.ContentLength != 0 {
			t.Fatalf("decode response: %v", err)
		}
	}
	return resp, out
}

func TestLLMKeySaveAndStatus(t *testing.T) {
	srv := newTestServer(t)

	// Not configured at boot.
	resp, err := srv.Client().Get(srv.URL + "/api/settings/llm-key")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var status map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["configured"] != false {
		t.Fatalf("configured = %v, want false before saving", status["configured"])
	}

	// Save with only the key — defaults fill base_url/model.
	resp, out := postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"api_key": "sk-super-secret-do-not-echo",
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("save status = %d, want 204 (body %v)", resp.StatusCode, out)
	}

	resp, err = srv.Client().Get(srv.URL + "/api/settings/llm-key")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["configured"] != true {
		t.Fatalf("configured = %v, want true after saving", status["configured"])
	}
}

func TestLLMKeyNeverEchoedBack(t *testing.T) {
	srv := newTestServer(t)
	secret := "sk-never-appears-in-any-response"

	if resp, _ := postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"api_key": secret, "base_url": "https://api.groq.com/openai/v1", "model": "llama-3.3-70b",
	}); resp.StatusCode != http.StatusNoContent {
		t.Fatal("save failed")
	}

	for _, path := range []string{"/api/settings/llm-key", "/api/settings/llm"} {
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		resp.Body.Close()
		if strings.Contains(string(buf[:n]), secret) {
			t.Errorf("GET %s echoed the API key back", path)
		}
	}
}

func TestLLMKeyValidation(t *testing.T) {
	srv := newTestServer(t)

	if resp, _ := postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"base_url": "https://api.openai.com/v1", "model": "gpt-4o-mini",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing api_key: status = %d, want 400", resp.StatusCode)
	}

	if resp, _ := postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"api_key": "sk-x", "base_url": "ftp://nope",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad scheme base_url: status = %d, want 400", resp.StatusCode)
	}
}

func TestLLMConfigReturnsNonSecretValues(t *testing.T) {
	srv := newTestServer(t)

	postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"api_key": "sk-x", "base_url": "https://api.groq.com/openai/v1", "model": "llama-3.3-70b",
	})

	resp, err := srv.Client().Get(srv.URL + "/api/settings/llm")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var cfg map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["configured"] != true {
		t.Errorf("configured = %v, want true", cfg["configured"])
	}
	if cfg["base_url"] != "https://api.groq.com/openai/v1" {
		t.Errorf("base_url = %v", cfg["base_url"])
	}
	if cfg["model"] != "llama-3.3-70b" {
		t.Errorf("model = %v", cfg["model"])
	}
}

func TestLLMPingAgainstFakeProvider(t *testing.T) {
	// Fake OpenAI-compatible provider standing in for the real one.
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/models" && r.Header.Get("Authorization") == "Bearer sk-good":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		case r.Header.Get("Authorization") == "Bearer sk-good":
			t.Errorf("unexpected provider path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"Incorrect API key"}}`))
		}
	}))
	defer provider.Close()

	srv := newTestServer(t)
	client := srv.Client()

	ping := func(t *testing.T) (bool, string) {
		t.Helper()
		resp, err := client.Post(srv.URL+"/api/settings/llm-ping", "application/json", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		ok, _ := out["ok"].(bool)
		msg, _ := out["error"].(string)
		return ok, msg
	}

	// Nothing saved yet → legible failure, not a crash.
	if ok, msg := ping(t); ok || msg == "" {
		t.Errorf("ping without key: ok=%v msg=%q", ok, msg)
	}

	// Bad key → provider error surfaced legibly.
	postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"api_key": "sk-bad", "base_url": provider.URL, "model": "test-model",
	})
	if ok, msg := ping(t); ok || !strings.Contains(msg, "Incorrect API key") {
		t.Errorf("ping with bad key: ok=%v msg=%q, want provider message", ok, msg)
	}

	// Good key → success.
	postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"api_key": "sk-good", "base_url": provider.URL, "model": "test-model",
	})
	if ok, msg := ping(t); !ok {
		t.Errorf("ping with good key failed: %q", msg)
	}
}

// Compile-time guard: the API depends on the llm.Client abstraction only.
var _ llm.Client = llm.NewOpenAIClient()
