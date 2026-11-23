package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/llm"
)

type pingLLMStub struct {
	err  error
	seen []llm.Config
}

func (p *pingLLMStub) Ping(_ context.Context, cfg llm.Config) error {
	p.seen = append(p.seen, cfg)
	return p.err
}
func (p *pingLLMStub) Complete(context.Context, llm.Config, llm.CompletionRequest) (string, error) {
	return "", errors.New("not used")
}

// call performs a JSON request and returns status, raw body and decoded body.
func call(t *testing.T, srv *httptest.Server, method, path string, payload any) (int, string, map[string]any) {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, _ := json.Marshal(payload)
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, srv.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, string(raw), out
}

func profilesOf(t *testing.T, srv *httptest.Server) []map[string]any {
	t.Helper()
	status, raw, out := call(t, srv, http.MethodGet, "/api/settings/llm/profiles", nil)
	if status != http.StatusOK {
		t.Fatalf("list status %d: %s", status, raw)
	}
	arr, ok := out["profiles"].([]any)
	if !ok {
		t.Fatalf("profiles must be an array, got %T (%s)", out["profiles"], raw)
	}
	res := make([]map[string]any, 0, len(arr))
	for _, a := range arr {
		res = append(res, a.(map[string]any))
	}
	return res
}

func TestPresetsAndProfilesWorkWithoutAnyProvider(t *testing.T) {
	srv := newTestServer(t)

	status, _, cat := call(t, srv, http.MethodGet, "/api/settings/llm/presets", nil)
	if status != http.StatusOK {
		t.Fatalf("presets status = %d", status)
	}
	presets, _ := cat["presets"].([]any)
	if len(presets) < 5 || cat["default_id"] == "" {
		t.Fatalf("catalog must be complete on a fresh install: %v", cat)
	}

	if got := profilesOf(t, srv); len(got) != 0 {
		t.Fatalf("fresh install must list zero profiles, got %v", got)
	}

	status, _, cfg := call(t, srv, http.MethodGet, "/api/settings/llm", nil)
	if status != http.StatusOK || cfg["configured"] != false {
		t.Fatalf("llm config on fresh install: %d %v", status, cfg)
	}

	// No key: friendly 200, never a 5xx.
	status, raw, ping := call(t, srv, http.MethodPost, "/api/settings/llm-ping", nil)
	if status != http.StatusOK || ping["ok"] != false || ping["code"] != llm.CodeNotConfigured {
		t.Fatalf("ping without key: %d %s", status, raw)
	}
}

func TestProfileLifecycleNeverLeaksKey(t *testing.T) {
	stub := &pingLLMStub{}
	srv := newTestServerWithLLM(t, stub)
	const secretA, secretB = "sk-secret-AAAA-1111", "sk-secret-BBBB-2222"

	// First profile becomes active by itself.
	status, raw, out := call(t, srv, http.MethodPost, "/api/settings/llm/profiles", map[string]any{
		"preset_id": "groq", "base_url": "https://api.groq.com/openai/v1",
		"model": "llama-3.3-70b-versatile", "api_key": secretA,
	})
	if status != http.StatusCreated {
		t.Fatalf("create A: %d %s", status, raw)
	}
	a := out["profile"].(map[string]any)
	if a["active"] != true || a["name"] != "Groq" {
		t.Fatalf("first profile must be active and named after its preset: %v", a)
	}
	idA := a["id"].(string)

	// The active profile is mirrored into the flat settings immediately.
	status, _, key := call(t, srv, http.MethodGet, "/api/settings/llm-key", nil)
	if status != http.StatusOK || key["configured"] != true {
		t.Fatalf("flat key not configured after first profile: %v", key)
	}

	// Second profile is created inactive.
	status, raw, out = call(t, srv, http.MethodPost, "/api/settings/llm/profiles", map[string]any{
		"name": "Meu OpenAI", "base_url": "https://api.openai.com/v1", "model": "gpt-4o-mini", "api_key": secretB,
	})
	if status != http.StatusCreated || out["profile"].(map[string]any)["active"] != false {
		t.Fatalf("create B: %d %s", status, raw)
	}
	idB := out["profile"].(map[string]any)["id"].(string)

	// Testing B pings with B's key without activating it.
	status, _, ping := call(t, srv, http.MethodPost, "/api/settings/llm/profiles/"+idB+"/test", nil)
	if status != http.StatusOK || ping["ok"] != true {
		t.Fatalf("test B: %d %v", status, ping)
	}
	if last := stub.seen[len(stub.seen)-1]; last.APIKey != secretB || last.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("test pinged the wrong config: %+v", last)
	}
	for _, p := range profilesOf(t, srv) {
		if p["id"] == idB && p["active"] == true {
			t.Fatal("testing a profile must not activate it")
		}
	}

	// Activating B makes tailoring/scoring/ping use it right away.
	status, raw, _ = call(t, srv, http.MethodPost, "/api/settings/llm/activate", map[string]string{"id": idB})
	if status != http.StatusOK {
		t.Fatalf("activate B: %d %s", status, raw)
	}
	_, _, ping = call(t, srv, http.MethodPost, "/api/settings/llm-ping", nil)
	if ping["ok"] != true {
		t.Fatalf("ping after activation: %v", ping)
	}
	if last := stub.seen[len(stub.seen)-1]; last.APIKey != secretB {
		t.Fatalf("active ping must use B's key, got %q", last.APIKey)
	}
	_, _, cfg := call(t, srv, http.MethodGet, "/api/settings/llm", nil)
	if cfg["model"] != "gpt-4o-mini" {
		t.Fatalf("flat model after activation = %v", cfg["model"])
	}

	// Cannot delete the active profile; can delete the other one.
	status, _, errBody := call(t, srv, http.MethodDelete, "/api/settings/llm/profiles/"+idB, nil)
	if status != http.StatusConflict || errBody["code"] != "profile_active_delete" {
		t.Fatalf("delete active: %d %v", status, errBody)
	}
	if status, _, _ = call(t, srv, http.MethodDelete, "/api/settings/llm/profiles/"+idA, nil); status != http.StatusNoContent {
		t.Fatalf("delete inactive: %d", status)
	}

	// Update keeps the stored key when api_key is omitted and re-syncs the
	// active profile.
	status, raw, _ = call(t, srv, http.MethodPut, "/api/settings/llm/profiles/"+idB, map[string]any{
		"name": "OpenAI novo modelo", "base_url": "https://api.openai.com/v1", "model": "gpt-4.1-mini",
	})
	if status != http.StatusOK {
		t.Fatalf("update B: %d %s", status, raw)
	}
	_, _, ping = call(t, srv, http.MethodPost, "/api/settings/llm-ping", nil)
	if last := stub.seen[len(stub.seen)-1]; ping["ok"] != true || last.APIKey != secretB || last.Model != "gpt-4.1-mini" {
		t.Fatalf("active profile edit must apply immediately and keep key: %+v", last)
	}

	// The secret never appears in any response body.
	for _, path := range []string{"/api/settings/llm/profiles", "/api/settings/llm", "/api/settings/llm-key", "/api/settings/llm/presets"} {
		_, body, _ := call(t, srv, http.MethodGet, path, nil)
		if strings.Contains(body, secretA) || strings.Contains(body, secretB) || strings.Contains(body, "sealed") {
			t.Fatalf("%s leaks key material: %s", path, body)
		}
	}
}

func TestProfileValidation(t *testing.T) {
	srv := newTestServer(t)

	status, _, body := call(t, srv, http.MethodPost, "/api/settings/llm/profiles", map[string]any{
		"base_url": "https://api.openai.com/v1", "model": "m",
	})
	if status != http.StatusBadRequest || body["code"] != "api_key_required" {
		t.Errorf("remote provider without key: %d %v", status, body)
	}

	status, _, body = call(t, srv, http.MethodPost, "/api/settings/llm/profiles", map[string]any{
		"base_url": "ftp://x", "api_key": "k",
	})
	if status != http.StatusBadRequest || body["code"] != "base_url_invalid" {
		t.Errorf("bad url: %d %v", status, body)
	}

	status, _, body = call(t, srv, http.MethodPost, "/api/settings/llm/profiles", map[string]any{
		"name": strings.Repeat("x", 200), "base_url": "https://a.example/v1", "api_key": "k",
	})
	if status != http.StatusBadRequest || body["code"] != "profile_name_invalid" {
		t.Errorf("long name: %d %v", status, body)
	}

	// Local runtimes (Ollama) need no key.
	status, raw, out := call(t, srv, http.MethodPost, "/api/settings/llm/profiles", map[string]any{
		"preset_id": "ollama", "base_url": "http://localhost:11434/v1", "model": "llama3.2",
	})
	if status != http.StatusCreated || out["profile"].(map[string]any)["local"] != true {
		t.Errorf("keyless local profile: %d %s", status, raw)
	}

	for _, c := range []struct{ method, path string }{
		{http.MethodPut, "/api/settings/llm/profiles/ghost"},
		{http.MethodDelete, "/api/settings/llm/profiles/ghost"},
		{http.MethodPost, "/api/settings/llm/profiles/ghost/test"},
	} {
		status, _, body := call(t, srv, c.method, c.path, map[string]any{"base_url": "https://a.example/v1", "api_key": "k"})
		if status != http.StatusNotFound || body["code"] != "profile_not_found" {
			t.Errorf("%s %s: %d %v", c.method, c.path, status, body)
		}
	}
	if status, _, _ := call(t, srv, http.MethodPost, "/api/settings/llm/activate", map[string]string{"id": "ghost"}); status != http.StatusNotFound {
		t.Errorf("activate ghost: %d", status)
	}
}

func TestLegacyKeyEndpointKeepsProfilesInSync(t *testing.T) {
	srv := newTestServer(t)

	// Legacy flat save on a fresh install creates the first active profile.
	if resp, _ := postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"api_key": "sk-legacy", "base_url": "https://api.groq.com/openai/v1", "model": "llama-3.3-70b-versatile",
	}); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("legacy save: %d", resp.StatusCode)
	}
	ps := profilesOf(t, srv)
	if len(ps) != 1 || ps[0]["active"] != true || ps[0]["model"] != "llama-3.3-70b-versatile" {
		t.Fatalf("profile not created from legacy save: %v", ps)
	}

	// A second legacy save updates the active profile instead of duplicating.
	if resp, _ := postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"api_key": "sk-legacy-2", "base_url": "https://api.openai.com/v1", "model": "gpt-4o-mini",
	}); resp.StatusCode != http.StatusNoContent {
		t.Fatal("second legacy save failed")
	}
	ps = profilesOf(t, srv)
	if len(ps) != 1 || ps[0]["model"] != "gpt-4o-mini" {
		t.Fatalf("legacy save must update the active profile: %v", ps)
	}
}
