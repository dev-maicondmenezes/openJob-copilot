package llm

import (
	"net/url"
	"strings"
	"testing"
)

func TestPresetsCatalogIsConsistent(t *testing.T) {
	cat := Presets()
	if cat.Version != PresetsVersion {
		t.Fatalf("version = %d, want %d", cat.Version, PresetsVersion)
	}
	seen := map[string]bool{}
	for _, p := range cat.Presets {
		if p.ID == "" || p.Name == "" {
			t.Errorf("preset %+v: id and name are required", p)
		}
		if seen[p.ID] {
			t.Errorf("duplicate preset id %q", p.ID)
		}
		seen[p.ID] = true

		u, err := url.Parse(p.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			t.Errorf("preset %s: invalid base_url %q", p.ID, p.BaseURL)
		}
		if strings.HasSuffix(p.BaseURL, "/") {
			t.Errorf("preset %s: base_url must not end with '/'", p.ID)
		}
		if p.RequiresKey == p.Local {
			t.Errorf("preset %s: a preset needs a key xor runs locally", p.ID)
		}
		if p.RequiresKey && !strings.HasPrefix(p.KeyURL, "https://") {
			t.Errorf("preset %s: needs an official https key_url", p.ID)
		}
		if p.Local != IsLocalBaseURL(p.BaseURL) {
			t.Errorf("preset %s: Local flag disagrees with IsLocalBaseURL(%q)", p.ID, p.BaseURL)
		}
		if p.FreeTier && p.FreeNote == "" {
			t.Errorf("preset %s: free tier must say what the limits are", p.ID)
		}
		if len(p.Models) == 0 {
			t.Errorf("preset %s: must expose at least one model for the selector", p.ID)
		}
		if len(p.Models) > 0 && p.Models[0] != p.Model {
			t.Errorf("preset %s: first model %q must equal the suggested default %q", p.ID, p.Models[0], p.Model)
		}
		if p.DocsURL != "" && !strings.HasPrefix(p.DocsURL, "https://") {
			t.Errorf("preset %s: docs_url must be https", p.ID)
		}
		if p.DocsURL == "" {
			t.Errorf("preset %s: docs_url is required for the provider help card", p.ID)
		}
	}
	def, ok := PresetByID(cat.DefaultID)
	if !ok || !def.FreeTier {
		t.Fatalf("default preset %q must exist and be free", cat.DefaultID)
	}
}

func TestPresetsKnownEndpoints(t *testing.T) {
	// Regression: the first F4 draft shipped ".../opensai/v1" for Groq.
	want := map[string]string{
		"gemini": "https://generativelanguage.googleapis.com/v1beta/openai",
		"groq":   "https://api.groq.com/openai/v1",
		"ollama": "http://localhost:11434/v1",
		"openai": "https://api.openai.com/v1",
	}
	for id, base := range want {
		p, ok := PresetByID(id)
		if !ok {
			t.Fatalf("missing preset %q", id)
		}
		if p.BaseURL != base {
			t.Errorf("%s base_url = %q, want %q", id, p.BaseURL, base)
		}
	}
	if p, _ := PresetByID("openai"); p.FreeTier {
		t.Error("openai must not be marked as free tier")
	}
}

func TestPresetsReturnsCopy(t *testing.T) {
	a := Presets()
	a.Presets[0].Name = "mutated"
	if Presets().Presets[0].Name == "mutated" {
		t.Fatal("Presets() must return an independent copy")
	}
}

func TestIsLocalBaseURL(t *testing.T) {
	cases := map[string]bool{
		"http://localhost:11434/v1":         true,
		"http://127.0.0.1:1234/v1":          true,
		"http://[::1]:11434/v1":             true,
		"http://host.docker.internal:11434": true,
		"https://api.openai.com/v1":         false,
		"https://localhost.evil.com/v1":     false,
		"http://127.0.0.1.evil.com/v1":      false,
		"":                                  false,
	}
	for in, want := range cases {
		if got := IsLocalBaseURL(in); got != want {
			t.Errorf("IsLocalBaseURL(%q) = %v, want %v", in, got, want)
		}
	}
}
